include .envrc

MIGRATIONS_DIR = cmd/migrate/migrations

.PHONY: migrate-create migrate-up migrate-down migrate-status run

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

# Chạy app
run:
	@go run cmd/api/*.go