// Package messagedaily appends each stored message as one JSON line to a file named by sent date (YYYY-MM-DD.jsonl).
package messagedaily

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/vietbui/chat-quality-agent/db/models"
)

var (
	mu      sync.Mutex
	root    string
	loc     = time.Local
	enabled bool
)

// Init sets the directory for daily JSONL files. Empty dir disables archiving.
func Init(dir string, location *time.Location) {
	mu.Lock()
	defer mu.Unlock()
	root = dir
	if location != nil {
		loc = location
	} else {
		loc = time.Local
	}
	enabled = dir != ""
}

// Append writes one JSON object per line to root/YYYY-MM-DD.jsonl based on m.SentAt.
func Append(m *models.Message) {
	if !enabled || m == nil {
		return
	}
	day := m.SentAt.In(loc).Format("2006-01-02")
	path := filepath.Join(root, day+".jsonl")

	line, err := json.Marshal(m)
	if err != nil {
		log.Printf("[messagedaily] marshal message %s: %v", m.ID, err)
		return
	}

	mu.Lock()
	defer mu.Unlock()

	if err := os.MkdirAll(root, 0755); err != nil {
		log.Printf("[messagedaily] mkdir %s: %v", root, err)
		return
	}

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Printf("[messagedaily] open %s: %v", path, err)
		return
	}
	defer f.Close()
	if _, err := f.Write(append(line, '\n')); err != nil {
		log.Printf("[messagedaily] write %s: %v", path, err)
	}
}
