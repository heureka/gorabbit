

# based on https://gist.github.com/prwhite/8168133
help: ## show this help
	@fgrep -h "##" $(MAKEFILE_LIST) | fgrep -v fgrep | sed -e 's/\\$$//' | sed -e 's/##//'


up: ## starts all dependencies for integration tests
	@docker compose up -d

test: up
	@RABBITMQ_URL=amqp://localhost:5672 go test -race -v -count=1 ./...
	@$(MAKE) down

down: ## stops all dependencies for integration tests
	@docker compose down