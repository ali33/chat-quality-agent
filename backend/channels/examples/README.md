# REST JSON channel — định dạng API

Kênh `rest_json` gọi **HTTP GET** tới API của bạn và đọc JSON cố định (xem các file `.example.json` cùng thư mục).

## Credentials (lưu trong kênh, đã mã hóa)

Xem [`credentials.example.json`](credentials.example.json).

- **`base_url`**: gốc URL (https, không bắt buộc dấu `/` cuối).
- **`list_conversations_path`**: đường dẫn danh sách hội thoại. CQA tự thêm query: `since` (RFC3339 UTC), `limit` (số nguyên).
- **`messages_path_template`**: đường dẫn tin nhắn; **bắt buộc** có chuỗi `{conversation_id}` (thay bằng `external_id` của hội thoại). CQA thêm query `since`.
- **`external_id`**: mã cố định cho nguồn này (trùng với `channels.external_id`, unique theo tenant + loại kênh).
- **`headers`**: (tuỳ chọn) header HTTP, ví dụ `Authorization`.
- **`insecure_skip_verify`**: chỉ dùng khi dev với HTTPS tự ký.
- **`timeout_seconds`**: mặc định 60.

## Response — danh sách hội thoại

GET `base_url` + `list_conversations_path?since=...&limit=...`

Body: **`{"conversations":[...]}`** hoặc **mảng JSON** `[...]`.

Xem [`conversations.response.example.json`](conversations.response.example.json).  
Có thể trả về **mảng gốc** (không bọc object): [`conversations.array.response.example.json`](conversations.array.response.example.json).

## Response — tin nhắn một hội thoại

GET `base_url` + path sau khi thay `{conversation_id}` + `?since=...`

Body: **`{"messages":[...]}`** hoặc **mảng** `[...]`.

Xem [`messages.response.example.json`](messages.response.example.json).

## Trường bắt buộc tối thiểu

| Phần | Trường |
|------|--------|
| Hội thoại | `external_id` |
| Tin nhắn | `external_id`, `sent_at` (khuyến nghị RFC3339) |

`sender_type`: `customer` \| `agent` \| `system` (mặc định `customer` nếu trống).  
`content_type`: mặc định `text`.
