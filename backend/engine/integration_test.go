// +build integration

package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"path/filepath"
	"testing"
	"time"

	"github.com/vietbui/chat-quality-agent/ai"
	"github.com/vietbui/chat-quality-agent/config"
	"github.com/vietbui/chat-quality-agent/db"
	"github.com/vietbui/chat-quality-agent/db/models"
	"github.com/vietbui/chat-quality-agent/pkg"
	"github.com/vietbui/chat-quality-agent/storage/messagedaily"
)

// SmartMockProvider analyzes the transcript and returns appropriate PASS/FAIL.
type SmartMockProvider struct{}

func (m *SmartMockProvider) AnalyzeChat(ctx context.Context, systemPrompt string, chatTranscript string) (ai.AIResponse, error) {
	// Simple heuristic: if transcript contains rude words → FAIL
	isRude := false
	rudeWords := []string{"Gi?", "Tu xem", "Khong biet", "De do"}
	for _, w := range rudeWords {
		if containsStr(chatTranscript, w) {
			isRude = true
			break
		}
	}

	var resp map[string]interface{}
	if isRude {
		resp = map[string]interface{}{
			"verdict": "FAIL",
			"score":   25,
			"review":  "Nhan vien tra loi coc loc, khong lich su.",
			"violations": []map[string]interface{}{
				{
					"severity":    "NGHIEM_TRONG",
					"rule":        "Chao hoi lich su",
					"evidence":    "NV: Gi?",
					"explanation": "Nhan vien khong chao hoi, tra loi thieu ton trong.",
					"suggestion":  "Nen bat dau bang loi chao than thien.",
				},
			},
			"summary": "Cuoc chat can cai thien nghiem tuc.",
		}
	} else {
		resp = map[string]interface{}{
			"verdict":    "PASS",
			"score":      90,
			"review":     "Nhan vien lich su, ho tro tot.",
			"violations": []interface{}{},
			"summary":    "Cuoc chat dat chuan.",
		}
	}

	respJSON, _ := json.Marshal(resp)
	return ai.AIResponse{
		Content:      string(respJSON),
		InputTokens:  200,
		OutputTokens: 100,
		Model:        "mock-model",
		Provider:     "mock",
	}, nil
}

func containsStr(s, substr string) bool {
	return len(s) >= len(substr) && findSubstr(s, substr)
}

func findSubstr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestIntegrationFullJobFlow(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("JWT_SECRET", "test-jwt-secret-at-least-32-chars-long")
	t.Setenv("ENCRYPTION_KEY", "12345678901234567890123456789012")
	t.Setenv("DB_DRIVER", "sqlite")
	t.Setenv("SQLITE_PATH", filepath.Join(tmpDir, "integration.db"))
	t.Setenv("MESSAGE_DATA_DIR", filepath.Join(tmpDir, "messages"))
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("config load: %v", err)
	}
	messagedaily.Init(cfg.MessageDataDir, cfg.MessageTimeLocation())

	if err := db.Connect(cfg); err != nil {
		t.Fatalf("connect database: %v", err)
	}
	defer db.Close()

	if err := db.AutoMigrate(); err != nil {
		t.Fatalf("AutoMigrate failed: %v", err)
	}

	tenantID := "inttest-" + pkg.NewUUID()[:8]
	channelID := "ch-inttest-" + pkg.NewUUID()[:8]
	now := time.Now()
	tCreate := now
	convBadID := "conv-int-bad-" + tenantID[:8]
	convGoodID := "conv-int-good-" + tenantID[:8]
	jobID := "job-inttest-" + tenantID[:8]
	channelIDsJSON, _ := json.Marshal([]string{channelID})

	// Defer order (last registered runs first): scrub job runs → job → messages → … → tenant.
	defer func() { db.DB.Where("id = ?", tenantID).Delete(&models.Tenant{}) }()
	defer func() { db.DB.Where("id = ?", channelID).Delete(&models.Channel{}) }()
	defer func() { db.DB.Where("tenant_id = ?", tenantID).Delete(&models.Conversation{}) }()
	defer func() { db.DB.Where("tenant_id = ?", tenantID).Delete(&models.Message{}) }()
	defer func() { db.DB.Where("id = ?", jobID).Delete(&models.Job{}) }()
	defer func() {
		var runIDs []string
		db.DB.Model(&models.JobRun{}).Where("job_id = ?", jobID).Pluck("id", &runIDs)
		for _, id := range runIDs {
			db.DB.Where("job_run_id = ?", id).Delete(&models.JobResult{})
			db.DB.Where("job_run_id = ?", id).Delete(&models.AIUsageLog{})
		}
		db.DB.Where("job_id = ?", jobID).Delete(&models.JobRun{})
	}()

	tenant := models.Tenant{
		ID:        tenantID,
		Name:      "Integration Test",
		Slug:      "inttest-" + tenantID[:8],
		Settings:  "{}",
		CreatedAt: tCreate,
		UpdatedAt: tCreate,
	}
	if err := db.DB.Create(&tenant).Error; err != nil {
		t.Fatalf("create tenant: %v", err)
	}

	ch := models.Channel{
		ID:                   channelID,
		TenantID:             tenantID,
		ChannelType:          "facebook",
		Name:                 "Test Channel",
		ExternalID:           "fake",
		CredentialsEncrypted: []byte{0},
		IsActive:             true,
		Metadata:             "{}",
		CreatedAt:            tCreate,
		UpdatedAt:            tCreate,
	}
	if err := db.DB.Create(&ch).Error; err != nil {
		t.Fatalf("create channel: %v", err)
	}

	convBad := models.Conversation{
		ID:                     convBadID,
		TenantID:               tenantID,
		ChannelID:              channelID,
		ExternalConversationID: "ext-bad",
		CustomerName:           "Khach Xau",
		LastMessageAt:          &now,
		MessageCount:           2,
		Metadata:               "{}",
		CreatedAt:              tCreate,
		UpdatedAt:              tCreate,
	}
	convGood := models.Conversation{
		ID:                     convGoodID,
		TenantID:               tenantID,
		ChannelID:              channelID,
		ExternalConversationID: "ext-good",
		CustomerName:           "Khach Tot",
		LastMessageAt:          &now,
		MessageCount:           2,
		Metadata:               "{}",
		CreatedAt:              tCreate,
		UpdatedAt:              tCreate,
	}
	if err := db.DB.Create(&convBad).Error; err != nil {
		t.Fatalf("create conv bad: %v", err)
	}
	if err := db.DB.Create(&convGood).Error; err != nil {
		t.Fatalf("create conv good: %v", err)
	}

	msgs := []models.Message{
		{
			ID:                pkg.NewUUID(),
			TenantID:          tenantID,
			ConversationID:    convBadID,
			ExternalMessageID: "m1",
			SenderType:        "customer",
			SenderName:        "Khach",
			Content:           "Xin chao",
			ContentType:       "text",
			Attachments:       "[]",
			SentAt:            now.Add(-5 * time.Minute),
			CreatedAt:         tCreate,
		},
		{
			ID:                pkg.NewUUID(),
			TenantID:          tenantID,
			ConversationID:    convBadID,
			ExternalMessageID: "m2",
			SenderType:        "agent",
			SenderName:        "NV",
			Content:           "Gi? Tu xem tren web di",
			ContentType:       "text",
			Attachments:       "[]",
			SentAt:            now.Add(-3 * time.Minute),
			CreatedAt:         tCreate,
		},
		{
			ID:                pkg.NewUUID(),
			TenantID:          tenantID,
			ConversationID:    convGoodID,
			ExternalMessageID: "m3",
			SenderType:        "customer",
			SenderName:        "Khach",
			Content:           "Chao ban",
			ContentType:       "text",
			Attachments:       "[]",
			SentAt:            now.Add(-5 * time.Minute),
			CreatedAt:         tCreate,
		},
		{
			ID:                pkg.NewUUID(),
			TenantID:          tenantID,
			ConversationID:    convGoodID,
			ExternalMessageID: "m4",
			SenderType:        "agent",
			SenderName:        "NV",
			Content:           "Xin chao! Em rat vui duoc ho tro. Anh can gi a?",
			ContentType:       "text",
			Attachments:       "[]",
			SentAt:            now.Add(-3 * time.Minute),
			CreatedAt:         tCreate,
		},
	}
	for i := range msgs {
		if err := db.DB.Create(&msgs[i]).Error; err != nil {
			t.Fatalf("create message: %v", err)
		}
	}

	jobRow := models.Job{
		ID:                jobID,
		TenantID:          tenantID,
		Name:              "QC Test Job",
		JobType:           "qc_analysis",
		InputChannelIDs:   string(channelIDsJSON),
		RulesContent:      "Nhan vien phai chao hoi lich su, tra loi day du.",
		RulesConfig:       "[]",
		ScheduleType:      "manual",
		IsActive:          true,
		Outputs:           "[]",
		CreatedAt:         tCreate,
		UpdatedAt:         tCreate,
	}
	if err := db.DB.Create(&jobRow).Error; err != nil {
		t.Fatalf("create job: %v", err)
	}

	var job models.Job
	if err := db.DB.First(&job, "id = ?", jobID).Error; err != nil {
		t.Fatalf("load job: %v", err)
	}

	analyzerCfg := &config.Config{}
	analyzer := NewAnalyzer(analyzerCfg)
	mockProvider := &SmartMockProvider{}

	run, err := analyzer.RunJobWithProvider(context.Background(), job, 3, mockProvider)
	if err != nil {
		t.Fatalf("RunJobWithProvider failed: %v", err)
	}

	if run.Status != "success" {
		t.Errorf("expected status 'success', got '%s' (error: %s)", run.Status, run.ErrorMessage)
	}

	var summary map[string]interface{}
	json.Unmarshal([]byte(run.Summary), &summary)
	log.Printf("Run summary: %s", run.Summary)

	analyzed := int(summary["conversations_analyzed"].(float64))
	passed := int(summary["conversations_passed"].(float64))
	issues := int(summary["issues_found"].(float64))

	if analyzed != 2 {
		t.Errorf("expected 2 conversations analyzed, got %d", analyzed)
	}
	if passed != 1 {
		t.Errorf("expected 1 passed (good conversation), got %d", passed)
	}
	if issues < 1 {
		t.Errorf("expected at least 1 issue (bad conversation), got %d", issues)
	}

	var results []models.JobResult
	db.DB.Where("job_run_id = ?", run.ID).Find(&results)

	evalCount := 0
	violationCount := 0
	for _, r := range results {
		switch r.ResultType {
		case "conversation_evaluation":
			evalCount++
		case "qc_violation":
			violationCount++
		}
	}

	if evalCount != 2 {
		t.Errorf("expected 2 conversation_evaluation records, got %d", evalCount)
	}
	if violationCount < 1 {
		t.Errorf("expected at least 1 qc_violation record, got %d", violationCount)
	}

	var usageLogs []models.AIUsageLog
	db.DB.Where("job_run_id = ?", run.ID).Find(&usageLogs)
	if len(usageLogs) != 2 {
		t.Errorf("expected 2 usage logs (1 per conversation), got %d", len(usageLogs))
	}
	for _, u := range usageLogs {
		if u.InputTokens != 200 || u.OutputTokens != 100 {
			t.Errorf("unexpected token counts: input=%d output=%d", u.InputTokens, u.OutputTokens)
		}
	}

	fmt.Printf("\n✅ Integration test PASSED:\n")
	fmt.Printf("   Conversations analyzed: %d\n", analyzed)
	fmt.Printf("   Passed: %d, Failed: %d\n", passed, analyzed-passed)
	fmt.Printf("   Violations found: %d\n", violationCount)
	fmt.Printf("   Usage logs: %d\n", len(usageLogs))
}
