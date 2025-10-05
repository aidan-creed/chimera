.PHONY: all up down logs build-backend run-backend migrate-up-platform migrate-down-platform migrate-up-demo migrate-down-demo migrate-up-claims migrate-down-claims migrate-up-all migrate-down-all install-frontend run-frontend db-reset

# ====================================================================================
# VARIABLES
# ====================================================================================
DB_USER=testuser
DB_PASSWORD=testpassword
DB_NAME=testdb
DB_HOST=localhost
DB_PORT=5432
DATABASE_URL="postgres://${DB_USER}:${DB_PASSWORD}@${DB_HOST}:${DB_PORT}/${DB_NAME}?sslmode=disable"

# ====================================================================================
# DOCKER COMMANDS
# ====================================================================================

## up: Starts all docker containers in detached mode
up:
	@echo "Bringing up Docker containers..."
	docker-compose up -d

## down: Stops containers and DESTROYS their data volumes
down:
	@echo "Bringing down Docker containers and removing volumes..."
	docker-compose down -v

## logs: Tails the logs of all running containers
logs:
	@echo "Tailing logs..."
	docker-compose logs -f

# ====================================================================================
# DATABASE COMMANDS
# ====================================================================================

## db-reset: DESTROYS the local database and starts a fresh one
db-reset: down
	@echo "Database destroyed. Starting fresh..."
	make up
	@echo "Waiting for DB to be ready..."
	sleep 5
	make migrate-up-all

## migrate-up-platform: Applies all PLATFORM database migrations
migrate-up-platform:
	@echo "Running PLATFORM database migrations up..."
	cd backend && go run github.com/pressly/goose/v3/cmd/goose -dir ./sql/platform/migrations postgres "${DATABASE_URL}" up

## migrate-down-platform: Rolls back the last PLATFORM database migration
migrate-down-platform:
	@echo "Rolling back last PLATFORM database migration..."
	cd backend && go run github.com/pressly/goose/v3/cmd/goose -dir ./sql/platform/migrations postgres "${DATABASE_URL}" down

## migrate-up-demo: Applies all DEMO database migrations
migrate-up-demo:
	@echo "Running DEMO database migrations up..."
	cd backend && go run github.com/pressly/goose/v3/cmd/goose -dir ./sql/apps/demo/migrations postgres "${DATABASE_URL}" up

## migrate-down-demo: Rolls back the last DEMO database migration
migrate-down-demo:
	@echo "Rolling back last DEMO database migration..."
	cd backend && go run github.com/pressly/goose/v3/cmd/goose -dir ./sql/apps/demo/migrations postgres "${DATABASE_URL}" down

## migrate-up-demo: Applies all CLAIMS database migrations
migrate-up-claims:
	@echo "Running CLAIMS database migrations up..."
	cd backend && go run github.com/pressly/goose/v3/cmd/goose -dir ./sql/apps/insurance/migrations postgres "${DATABASE_URL}" up

## migrate-down-demo: Rolls back the last CLAIMS database migration
migrate-down-claims:
	@echo "Rolling back last CLAIMS database migration..."
	cd backend && go run github.com/pressly/goose/v3/cmd/goose -dir ./sql/apps/insurance/migrations postgres "${DATABASE_URL}" down


## migrate-up-all: Applies ALL database migrations (platform then demo)
migrate-up-all: migrate-up-platform migrate-up-demo migrate-up-claims
	@echo "All migrations applied."

## migrate-down-all: Rolls back the last migration from ANY directory
migrate-down-all:
	@echo "Rolling back last CLAIMS migration (if any)..."
	cd backend && go run github.com/pressly/goose/v3/cmd/goose -dir ./sql/apps/insurance/migrations postgres "${DATABASE_URL}" down || true
	@echo "Rolling back last DEMO migration (if any)..."
	cd backend && go run github.com/pressly/goose/v3/cmd/goose -dir ./sql/apps/demo/migrations postgres "${DATABASE_URL}" down || true
	@echo "Rolling back last PLATFORM migration (if any)..."
	cd backend && go run github.com/pressly/goose/v3/cmd/goose -dir ./sql/platform/migrations postgres "${DATABASE_URL}" down || true


# ====================================================================================
# BACKEND COMMANDS (Go)
# ====================================================================================

## build-backend: Compiles the Go backend application
build-backend:
##	@echo "Building backend..."
	cd backend && go build -o ../chimera-server ./cmd/server

## run-backend: Runs the compiled Go backend application
run-backend: build-backend
##	@echo "Running backend server..."
	./chimera-server

# ====================================================================================
# FRONTEND COMMANDS (Node)
# ====================================================================================

## install-frontend: Installs frontend dependencies
install-frontend:
	@echo "Installing frontend dependencies..."
	cd frontend && npm install

## run-frontend: Starts the frontend development server
run-frontend:
	@echo "Starting frontend dev server..."
	cd frontend && npm run dev

# ====================================================================================
# ALL-IN-ONE
# ====================================================================================

## all: Starts the database, applies all migrations, and runs both backend and frontend
all: up migrate-up-all run-backend run-frontend
