include .env

build:
	tailwindcss --minify -i ./internals/web/tailwind/tailwind_input.css -o ./internals/web/assets/css/style.css
	sqlc generate
	go build -o ./bin/server ./cmd/server/

run: build
	./bin/server

clean:
	rm -rf ./bin/

WATCH_CMD = watchexec --ignore-nothing
watch:
	@tailwindcss --watch --minify -i ./internals/web/tailwind/tailwind_input.css -o ./internals/web/assets/css/style.css &
	# @$(WATCH_CMD) -w ./internals/web/tailwind_input.css \
	# 							-w ./internals/web/assets/js/   -e js \
	# 							-w ./internals/web/templates/   -e go \
	# 							-w ./internals/handlers/        -e go \
	# 							-- tailwindcss --minify -i ./internals/web/tailwind_input.css -o ./internals/web/assets/css/style.css &
	@$(WATCH_CMD) -w ./internals/services/ -e sql -- sqlc generate &
	@$(WATCH_CMD) -r --stop-timeout=0        \
								-w ./internals/      -e go \
								-w ./cmd/server/     -e go \
								-w ./internals/web/assets/ \
								-- "go build -o ./bin/server ./cmd/server/ && ./bin/server"

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
	@$(GOOSE_ENV) goose up
goose-down: validate-goose-service
	@$(GOOSE_ENV) goose down
goose-reset: validate-goose-service
	@$(GOOSE_ENV) goose reset
goose-new: validate-goose-service validate-goose-name
	@$(GOOSE_ENV) goose create -s $(name) sql

pg-cli:
	@echo  "$(POSTGRES_URL)"
	@pgcli "$(POSTGRES_URL)"

redis-cli:
	@valkey-cli -p "$(REDIS_PORT)"
