# Không bắt buộc: clone mới chưa có .envrc vẫn chạy được build/run/gen-docs.
# Các lệnh migrate-* và seed thì cần nó (cp .envrc.example .envrc).
-include .envrc

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
	@goose -dir $(MIGRATIONS_DIR) postgres "$(DB_ADDR)" up

# Rollback 1 bước
migrate-down:
	@goose -dir $(MIGRATIONS_DIR) postgres "$(DB_ADDR)" down

# Xem trạng thái migration	
migrate-status:
	@goose -dir $(MIGRATIONS_DIR) postgres "$(DB_ADDR)" status

# Thêm dữ liệu mẫu (giữ nguyên dữ liệu cũ)
seed:
	@DB_ADDR=$(DB_ADDR) go run ./cmd/migrate/seed -users=$(USERS) -posts=$(POSTS) -comments=$(COMMENTS)

# Xoá sạch users/posts/comments rồi seed lại từ đầu
seed-reset:
	@DB_ADDR=$(DB_ADDR) go run ./cmd/migrate/seed -reset -users=$(USERS) -posts=$(POSTS) -comments=$(COMMENTS)

# Chạy app
run: gen-docs
	@go run cmd/api/*.go

# Build binary vào bin/ (sinh swagger docs trước, vì docs/ không được commit)
build: gen-docs
	@go build -o ./bin/main ./cmd/api


# Sinh swagger docs vào docs/. swag được ghim trong go.mod (go tool), không cần cài riêng
gen-docs:
	@go tool swag init -g ./api/main.go -d cmd,internal && go tool swag fmt