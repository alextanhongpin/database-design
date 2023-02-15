include .env
export

up:
	@docker-compose up -d

down:
	@docker-compose down


compile:
	@go build -o=./cmd/main --tags "fts5" ./cmd/main.go

init:
	@./cmd/main init

index:
	@./cmd/main index

search:
	@./cmd/main search -q ${q}
