include .env

build:
	sqlc generate
	go fmt ./...
	go build -o ./bin/server ./cmd/server/

run: build
	./bin/server

clean:
	rm -rf ./bin/

validate-goose-service:
	@if [ -z "$(service)" ]; then \
		echo "error: 'service' is required."; \
		echo "usage: make goose-{up,down,reset,new} service=<service>"; \
		exit 1; \
	fi
validate-goose-name:
	@if [ -z "$(name)" ]; then \
		echo "error: 'name' is required for 'new' command."; \
		echo "usage: make goose-new service=<service> name=<name>"; \
		exit 1; \
	fi
GOOSE_ENV = GOOSE_DRIVER="postgres" GOOSE_DBSTRING="$(POSTGRES_URL)" GOOSE_MIGRATION_DIR="./internals/services/$(service)/db/migrations/"
goose-up: validate-goose-service
	@$(GOOSE_ENV) goose up -table="$(service)_goose_db_version"
goose-down: validate-goose-service
	@$(GOOSE_ENV) goose down -table="$(service)_goose_db_version"
goose-reset: validate-goose-service
	@$(GOOSE_ENV) goose reset -table="$(service)_goose_db_version"
goose-new: validate-goose-service validate-goose-name
	@$(GOOSE_ENV) goose create $(name) sql -table="$(service)_goose_db_version"

pg-cli:
	@echo  "$(POSTGRES_URL)"
	@pgcli "$(POSTGRES_URL)"

redis-cli:
	@redis-cli -p "$(REDIS_PORT)"
