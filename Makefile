include .env
export

cli := ./cmd/cli/main

up:
	@docker-compose up -d


down:
	@docker-compose down


compile:
	@go build -o=${cli} --tags "fts5" ./cmd/cli/main.go


init:
	@${cli} init


index:
	@${cli} index


search:
	@${cli} search -q ${q}


server:
	@go build -o=./cmd/server/main --tags "fts5" ./cmd/server/main.go
	@./cmd/server/main
