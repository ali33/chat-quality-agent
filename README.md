# Chat Quality Agent (CQA)

[![Docker Hub](https://img.shields.io/docker/v/buitanviet/chat-quality-agent?label=Docker%20Hub&sort=semver)](https://hub.docker.com/r/buitanviet/chat-quality-agent)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

Hệ thống phân tích chất lượng chăm sóc khách hàng bằng AI. Tự động đồng bộ tin nhắn từ Zalo OA, Facebook Messenger, dùng AI (Claude/Gemini) đánh giá chất lượng CSKH và gửi cảnh báo qua Telegram/Email.

📖 **Hướng dẫn sử dụng chi tiết: [https://tanviet12.github.io/chat-quality-agent/](https://tanviet12.github.io/chat-quality-agent/)**

📄 **Thay đổi so với bản gốc (fork): [UPDATE.md](UPDATE.md)**

![Dashboard](https://raw.githubusercontent.com/tanviet12/chat-quality-agent/main/docs/public/screenshots/dashboard.png)

## Tính năng

- **Đồng bộ tin nhắn** từ Zalo OA và Facebook Messenger
- **Đánh giá chất lượng CSKH** bằng AI (Claude hoặc Gemini) — Đạt/Không đạt, điểm 0-100, nhận xét chi tiết
- **Phân loại chat** theo chủ đề tùy chỉnh (khiếu nại, góp ý, hỏi giá...)
- **Cảnh báo tự động** qua Telegram và Email
- **Batch AI mode** — gom nhiều cuộc chat/lần gọi AI, tiết kiệm chi phí
- **Dashboard** với biểu đồ, thống kê, cảnh báo gần đây
- **Multi-tenant** — nhiều công ty trên 1 hệ thống, phân quyền Owner > Admin > Member
- **Tích hợp MCP** cho Claude Web/Desktop
- **SSL tự động** qua Let's Encrypt (tùy chọn)

## Cài đặt nhanh

### Cách 1: Cài tự động (khuyến nghị)

```bash
curl -s https://raw.githubusercontent.com/tanviet12/chat-quality-agent/main/install.sh | sudo bash
```

Script tự cài Docker, tạo secrets ngẫu nhiên, pull images và khởi chạy.

### Cách 2: Build từ source

```bash
git clone https://github.com/tanviet12/chat-quality-agent.git
cd chat-quality-agent
cp .env.example .env
# Sửa .env
docker compose up -d --build
```

Truy cập: **http://your-server-ip:8080** (hoặc `http://localhost:8080`) — Lần đầu sẽ hiện trang Setup để tạo tài khoản admin. (Cổng có thể đổi qua `SERVER_PORT` trong `.env`.)

### Bật SSL (tùy chọn)

Trong bản fork này, compose mặc định **không** gồm Nginx. Có thể dùng **Caddy**, **Traefik** hoặc tự triển khai Nginx + Let’s Encrypt phía trước cổng **8080**, hoặc tham khảo image `docker/Dockerfile.nginx` trong repo gốc. Chi tiết khác biệt: [UPDATE.md](UPDATE.md).

## Công nghệ

| Thành phần | Công nghệ |
|-----------|-----------|
| Backend | Go 1.25+ / Gin |
| Frontend | Vue 3 + Vuetify 4 + Vite |
| Database | SQLite mặc định; tùy chọn MySQL / PostgreSQL / SQL Server — [chi tiết](UPDATE.md) |
| AI | Claude (Anthropic) / Gemini (Google) |
| Reverse Proxy | Tùy chọn (Caddy/Nginx/…) phía ngoài cổng 8080 — [chi tiết](UPDATE.md) |
| Deploy | Docker Compose (mặc định: một service app) |

## Kiến trúc (tóm tắt fork)

```
  Internet ────────> (tùy chọn: reverse proxy SSL) ────────> CQA App :8080
                                                                    │
                    SQLite file / hoặc MySQL · Postgres · SQL Server
```

Bản gốc dùng Nginx + MySQL trong Compose; bản hiện tại: [UPDATE.md](UPDATE.md).

## Cấu trúc dự án

```
chat-quality-agent/
├── backend/            # Go API server
│   ├── ai/             # AI providers (Claude, Gemini)
│   ├── api/            # REST API handlers + middleware
│   ├── channels/       # Zalo OA, Facebook adapters
│   ├── db/             # GORM models + đa driver SQL
│   ├── engine/         # Analyzer + Sync + Scheduler
│   ├── mcp/            # MCP server cho Claude
│   └── notifications/  # Telegram + Email
├── frontend/           # Vue 3 SPA
├── docker/             # Nginx + SSL configs
├── docs/               # Tài liệu hướng dẫn (VitePress)
├── docker-compose.yml      # Build từ source
├── docker-compose.hub.yml  # Dùng image Docker Hub
└── Dockerfile
```

## Hướng dẫn sử dụng

1. **Kết nối kênh chat**: Cài đặt > Kênh chat > Kết nối Facebook/Zalo
2. **Đồng bộ tin nhắn**: Bấm "Đồng bộ ngay" hoặc chờ tự động
3. **Cấu hình AI**: Cài đặt > AI > Chọn Claude/Gemini + nhập API key
4. **Tạo công việc**: Công việc > Tạo mới > Wizard 6 bước
5. **Chạy phân tích**: Chi tiết công việc > Chạy thử hoặc Chạy ngay
6. **Xem kết quả**: Chi tiết công việc > Kết quả đánh giá

## Biến môi trường

| Biến | Mô tả | Bắt buộc |
|------|-------|----------|
| `JWT_SECRET` | Secret cho JWT tokens (min 32 ký tự) | Có |
| `ENCRYPTION_KEY` | Key 32 bytes cho AES-256-GCM | Có |
| `DB_DRIVER` | `sqlite` / `mysql` / `postgres` / `sqlserver` | Không (mặc định sqlite) |
| `SQLITE_PATH`, `MESSAGE_DATA_DIR` | File DB và thư mục JSONL tin nhắn theo ngày | Theo `.env.example` |
| `DB_*` | Host, port, user, password, DB name khi không dùng SQLite | Khi dùng server DB |
| `LEGO_DOMAIN` | Domain cho SSL tự động (nếu dùng stack Nginx+Lego riêng) | Không |
| `LEGO_EMAIL` | Email cho Let's Encrypt | Không |
| `APP_URL` | URL công khai (cho links notification) | Không |

Xem đầy đủ trong [.env.example](.env.example). Khác biệt so với bản gốc: [UPDATE.md](UPDATE.md).

## Screenshots

| | |
|---|---|
| ![Setup](https://raw.githubusercontent.com/tanviet12/chat-quality-agent/main/docs/public/screenshots/setup.png) | ![Dashboard](https://raw.githubusercontent.com/tanviet12/chat-quality-agent/main/docs/public/screenshots/dashboard.png) |
| Trang Setup lần đầu | Dashboard |
| ![Kết nối kênh](https://raw.githubusercontent.com/tanviet12/chat-quality-agent/main/docs/public/screenshots/ket-noi-kenh-chat.png) | ![Tạo công việc](https://raw.githubusercontent.com/tanviet12/chat-quality-agent/main/docs/public/screenshots/tao-cong-viec.png) |
| Kết nối kênh chat | Tạo công việc |
| ![Kết quả QC](https://raw.githubusercontent.com/tanviet12/chat-quality-agent/main/docs/public/screenshots/ket-qua-cong-viec-danh-gia.png) | ![Kết quả phân loại](https://raw.githubusercontent.com/tanviet12/chat-quality-agent/main/docs/public/screenshots/ket-qua-cong-viec-phan-loai.png) |
| Kết quả đánh giá QC | Kết quả phân loại |
| ![Chi tiết tin nhắn](https://raw.githubusercontent.com/tanviet12/chat-quality-agent/main/docs/public/screenshots/chi-tiet-tin-nhan-va-danh-gia.png) | ![Chi tiết kênh](https://raw.githubusercontent.com/tanviet12/chat-quality-agent/main/docs/public/screenshots/chi-tiet-kenh-chat.png) |
| Chi tiết tin nhắn + đánh giá | Chi tiết kênh chat |

## Changelog

- **So với repo gốc (upstream):** **[UPDATE.md](UPDATE.md)**
- **Lịch sử phiên bản theo tag:** **[CHANGELOG.md](CHANGELOG.md)**

## Tài liệu

Xem tài liệu chi tiết tại: **[https://tanviet12.github.io/chat-quality-agent/](https://tanviet12.github.io/chat-quality-agent/)**

## License

[MIT](LICENSE) - SePay
