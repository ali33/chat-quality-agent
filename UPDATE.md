# Cập nhật so với bản gốc

**Tóm lược — fork này hỗ trợ:**

- **Nhiều loại database** qua `DB_DRIVER`: **SQLite** (mặc định, file đơn giản), **MySQL**, **PostgreSQL**, **SQL Server** (và alias như `mssql`, `mariadb`…). Chi tiết biến môi trường ở **mục 1** và `.env.example`.
- **Build thành file thực thi `.exe` trên Windows**: chạy `build-backend.bat` để có `build\cqa-server.exe` (SQLite dùng **glebarez** — pure Go, **không cần gcc/CGO**). Có giao diện web: `build-frontend.bat` rồi `build-backend.bat` sẽ copy `frontend\dist` vào `build\static` (hoặc set `STATIC_DIR`). Chi tiết ở **mục 5**.

---

Tài liệu này mô tả các thay đổi của nhánh/fork hiện tại so với **bản gốc** [tanviet12/chat-quality-agent](https://github.com/tanviet12/chat-quality-agent) (MySQL + Nginx + Docker Compose ba service).

---

## 1. Cơ sở dữ liệu: chọn backend qua cấu hình

- Thêm **`DB_DRIVER`**: `sqlite` (mặc định), `mysql`, `postgres`, `sqlserver` (có alias như `mssql`, `postgresql`, `mariadb`…).
- **SQLite**: `SQLITE_PATH` (mặc định `data/cqa.db`).
- **MySQL / PostgreSQL / SQL Server**: `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, `DB_NAME`; Postgres thêm `DB_SSLMODE`.
- Lớp kết nối tại `backend/db/database.go`: chọn dialector GORM theo `DB_DRIVER`, pool phù hợp (SQLite `MaxOpenConns=1`).
- Ràng buộc unique và biểu thức ngày trong dashboard xử lý theo từng engine (`db.DateSQL`, index/migration tương thích).
- Model: trường nhị phân (credentials, settings) dùng kiểu **BLOB** tương thích đa DB.

---

## 2. Docker & triển khai

- **`docker-compose.yml`**: một service **`app`**, publish **`8080`**, volume `cqa_data:/var/lib/cqa` (SQLite, thư mục tin nhắn theo ngày, file đính kèm).
- **Bỏ** service **MySQL** và **Nginx** khỏi compose mặc định; ứng dụng Go phục vụ trực tiếp HTTP (API + static production).
- **`docker-compose.hub.yml`**: cùng hướng (app + Watchtower, không MySQL/Nginx).
- **`Dockerfile`**: build backend với **`CGO_ENABLED=0`** (SQLite **glebarez**), không cần `gcc` trên Alpine.
- **`.github/workflows/release.yml`**: bỏ bước build/push image **nginx**; chỉ build image ứng dụng.

HTTPS / reverse proxy: đặt **Caddy, Traefik, Nginx**… phía trước cổng 8080 nếu cần.

---

## 3. File tin nhắn theo ngày (JSONL)

- Package **`storage/messagedaily`**: mỗi lần lưu tin (đồng bộ kênh, cập nhật attachment, import demo) ghi **một dòng JSON** vào  
  `{MESSAGE_DATA_DIR}/YYYY-MM-DD.jsonl`  
  (ngày theo `sent_at`, timezone từ `TZ` nếu có).
- SQLite/DB quan hệ vẫn là nguồn cho API; JSONL phục vụ backup/phân tích ngoài.

---

## 4. HTTP server (Go)

- **`main.go`**: khởi động bằng **`http.ListenAndServe(addr, router)`** thay cho `gin.Run` (Gin vẫn là router, implement `http.Handler`).

---

## 5. Cấu hình & chạy local (Windows)

- **`build-frontend.bat`**: `npm ci` + build Vite, copy `frontend/dist` → `build/static`.
- **`build-backend.bat`**: `go mod tidy` + `go build` → `build/cqa-server.exe` với **`CGO_ENABLED=0`** (SQLite **github.com/glebarez/sqlite**, không cần MSYS2/gcc). Nếu đã có `frontend\dist\index.html`, script **tự copy** vào `build\static\` cạnh exe.
- Production: static mặc định là **`<thư-mục-chứa-exe>/static`** (hoặc env **`STATIC_DIR`**); xem `backend/config/config.go` và `api/router.go`.
- File **`build-windows.bat`** đã **tách** thành hai script trên.
- **`github.com/joho/godotenv`**: tự đọc **`.env`** (thư mục làm việc, sau đó cùng thư mục với `.exe`) **trước** `config.Load()`; biến môi trường đã có sẵn **không** bị ghi đè.
- **`.env.example`**, **`install.sh`**: cập nhật theo `DB_DRIVER`, SQLite, `MESSAGE_DATA_DIR` (không còn bắt buộc `DB_PASSWORD` / MySQL khi dùng SQLite).

---

## 6. Kiểm thử & module Go

- **`engine/integration_test.go`**: seed dữ liệu bằng **GORM** thay vì SQL thuần (`NOW()`, `X'00'`) để tương thích đa driver; mặc định SQLite trong thư mục tạm.
- **`go.mod`**: thêm driver GORM **mysql**, **postgres**, **sqlserver**; giữ **sqlite**; thêm **godotenv**.
- Chạy **`go mod tidy`** trong `backend` sau khi pull (script `build-backend.bat` đã gọi sẵn).

---

## 7. Git & thư mục build

- **`.gitignore`**: thêm `build/` (output Windows), giữ `data/` cho SQLite/JSONL local.

---

## Tóm tắt nhanh

| Hạng mục | Bản gốc (tham chiếu) | Bản hiện tại |
|----------|----------------------|--------------|
| DB mặc định | MySQL 8 trong Compose | SQLite file + tùy chọn MySQL/Postgres/SQL Server |
| Compose | app + nginx + db | Chủ yếu **app** (+ volume) |
| Tin nhắn | Chỉ trong bảng DB | DB + file **JSONL theo ngày** |
| Chạy HTTP | Gin `Run` / qua Nginx | `net/http` + Go phục vụ static |
| `.env` khi chạy exe | Không tự load | **godotenv** (cwd + cạnh exe) |
| Build Windows | (không có sẵn) | `build-frontend.bat` + `build-backend.bat` → **`cqa-server.exe`** + `static\`, không cần gcc |

Nếu bạn merge ngược về upstream, cần rà soát xung đột ở `docker-compose*.yml`, `Dockerfile`, `README`, docs và workflow release.
