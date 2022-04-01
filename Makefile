include .env
export

up:
	@docker-compose up -d

down:
	@docker-compose down
