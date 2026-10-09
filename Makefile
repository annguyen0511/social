# KHÔNG dùng `include .envrc`. make đọc file theo cú pháp của nó, nên
# export FOO="bar" cho ra giá trị kèm nguyên cặp dấu nháy, rồi `export` đẩy
# chuỗi hỏng đó xuống mọi tiến trình con và ghi đè giá trị đúng của direnv.
# Hậu quả: chuỗi kết nối DB không parse được và pq báo "SSL is not enabled".
#
# Never `include .envrc`. make parses it with its own syntax, so
# export FOO="bar" keeps the quotes inside the value, and `export` then pushes
# that broken string into every child process, overriding what direnv loaded.
#
# direnv đã nạp .envrc vào shell rồi; ENV_SH chỉ là phương án dự phòng cho máy
# không dùng direnv. Nạp bằng shell nên dấu nháy được xử lý đúng.
ENV_SH = set -a; [ -f .envrc ] && . ./.envrc || true; set +a;

MIGRATIONS_DIR = cmd/migrate/migrations

.PHONY: migrate-create migrate-up migrate-down migrate-status run build seed seed-reset gen-docs

# Số lượng bản ghi seed, ghi đè được: make seed USERS=10 POSTS=20 COMMENTS=50
USERS ?= 100
POSTS ?= 200
COMMENTS ?= 500

# Tạo migration mới: make migrate-create name=create_posts_table
migrate-create:
	@goose -dir $(MIGRATIONS_DIR) create $(name) sql -s

# Chạy migration lên
migrate-up:
	@$(ENV_SH) goose -dir $(MIGRATIONS_DIR) postgres "$$DB_ADDR" up

# Rollback 1 bước
migrate-down:
	@$(ENV_SH) goose -dir $(MIGRATIONS_DIR) postgres "$$DB_ADDR" down

# Xem trạng thái migration	
migrate-status:
	@$(ENV_SH) goose -dir $(MIGRATIONS_DIR) postgres "$$DB_ADDR" status

# Thêm dữ liệu mẫu (giữ nguyên dữ liệu cũ)
seed:
	@$(ENV_SH) go run ./cmd/migrate/seed -users=$(USERS) -posts=$(POSTS) -comments=$(COMMENTS)

# Xoá sạch users/posts/comments rồi seed lại từ đầu
seed-reset:
	@$(ENV_SH) go run ./cmd/migrate/seed -reset -users=$(USERS) -posts=$(POSTS) -comments=$(COMMENTS)

# Chạy app. Build rồi exec thay vì `go run`: `go run` sinh một binary tạm làm
# tiến trình con, và khi `go run` bị giết thì binary con thường sống sót, giữ
# nguyên cổng 8080 và khiến lần chạy sau báo "address already in use". `exec`
# thay thế hẳn shell bằng binary nên tín hiệu tới thẳng nó, không còn mồ côi.
#
# Build then exec instead of `go run`: `go run` compiles to a temporary binary
# and runs it as a child, which survives when `go run` is killed and keeps
# port 8080, so the next run fails with "address already in use". `exec`
# replaces the shell with the binary, so signals reach it and nothing is
# orphaned.
run: gen-docs
	@$(ENV_SH) go build -o ./bin/main ./cmd/api && exec ./bin/main

# Build binary vào bin/ (sinh swagger docs trước, vì docs/ không được commit)
build: gen-docs
	@go build -o ./bin/main ./cmd/api


# Sinh swagger docs vào docs/. swag được ghim trong go.mod (go tool), không cần cài riêng
gen-docs:
	@go tool swag init -g ./api/main.go -d cmd,internal && go tool swag fmt
