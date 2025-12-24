.PHONY: help build run test clean docker-build-app docker-build-docreader docker-build-frontend docker-build-all docker-run migrate-up migrate-down docker-restart docker-stop start-all stop-all start-ollama stop-ollama build-images build-images-app build-images-docreader build-images-frontend clean-images check-env list-containers pull-images show-platform dev-start dev-stop dev-restart dev-logs dev-status dev-app dev-frontend docs install-swagger

# Show help
help:
	@echo "WeKnora Makefile Help"
	@echo ""
	@echo "Basic commands:"
	@echo "  build             Build the application"
	@echo "  run               Run the application"
	@echo "  test              Run tests"
	@echo "  clean             Clean build artifacts"
	@echo ""
	@echo "Docker commands:"
	@echo "  docker-build-app       Build the application Docker image (wechatopenai/weknora-app)"
	@echo "  docker-build-docreader Build the docreader image (wechatopenai/weknora-docreader)"
	@echo "  docker-build-frontend  Build the frontend image (wechatopenai/weknora-ui)"
	@echo "  docker-build-all       Build all Docker images"
	@echo "  docker-run            Run Docker containers"
	@echo "  docker-stop           Stop Docker containers"
	@echo "  docker-restart        Restart Docker containers"
	@echo ""
	@echo "Service management:"
	@echo "  start-all         Start all services"
	@echo "  stop-all          Stop all services"
	@echo "  start-ollama      Start only the Ollama service"
	@echo ""
	@echo "Image builds:"
	@echo "  build-images      Build all images from source"
	@echo "  build-images-app  Build the app image from source"
	@echo "  build-images-docreader Build the docreader image from source"
	@echo "  build-images-frontend  Build the frontend image from source"
	@echo "  clean-images      Clean local images"
	@echo ""
	@echo "Database:"
	@echo "  migrate-up        Run database migrations"
	@echo "  migrate-down      Roll back database migrations"
	@echo ""
	@echo "Developer tools:"
	@echo "  fmt               Format code"
	@echo "  lint              Run the linter"
	@echo "  deps              Download dependencies"
	@echo "  docs              Generate Swagger API documentation"
	@echo "  install-swagger   Install the swag tool"
	@echo ""
	@echo "Environment checks:"
	@echo "  check-env         Check environment configuration"
	@echo "  list-containers   List running containers"
	@echo "  pull-images       Pull the latest images"
	@echo "  show-platform     Show the current build platform"
	@echo ""
	@echo "Development mode (recommended):"
	@echo "  dev-start         Start the development infrastructure (dependencies only)"
	@echo "  dev-stop          Stop the development environment"
	@echo "  dev-restart       Restart the development environment"
	@echo "  dev-logs          View development logs"
	@echo "  dev-status        Show development status"
	@echo "  dev-app           Run the backend locally (run dev-start first)"
	@echo "  dev-frontend      Run the frontend locally (run dev-start first)"

# Go related variables
BINARY_NAME=WeKnora
MAIN_PATH=./cmd/server

# Docker related variables
DOCKER_IMAGE=wechatopenai/weknora-app
DOCKER_TAG=latest

# Platform detection
ifeq ($(shell uname -m),x86_64)
    PLATFORM=linux/amd64
else ifeq ($(shell uname -m),aarch64)
    PLATFORM=linux/arm64
else ifeq ($(shell uname -m),arm64)
    PLATFORM=linux/arm64
else
    PLATFORM=linux/amd64
endif

# Build the application
build:
	go build -o $(BINARY_NAME) $(MAIN_PATH)

# Run the application
run: build
	./$(BINARY_NAME)

# Run tests
test:
	go test -v ./...

# Clean build artifacts
clean:
	go clean
	rm -f $(BINARY_NAME)

# Build Docker image
docker-build-app:
	@echo "Retrieving version information..."
	@eval $$(./scripts/get_version.sh env); \
	./scripts/get_version.sh info; \
	docker build --platform $(PLATFORM) \
		--build-arg VERSION_ARG="$$VERSION" \
		--build-arg COMMIT_ID_ARG="$$COMMIT_ID" \
		--build-arg BUILD_TIME_ARG="$$BUILD_TIME" \
		--build-arg GO_VERSION_ARG="$$GO_VERSION" \
		-f docker/Dockerfile.app -t $(DOCKER_IMAGE):$(DOCKER_TAG) .

# Build docreader Docker image
docker-build-docreader:
	docker build --platform $(PLATFORM) -f docker/Dockerfile.docreader -t wechatopenai/weknora-docreader:latest .

# Build frontend Docker image
docker-build-frontend:
	docker build --platform $(PLATFORM) -f frontend/Dockerfile -t wechatopenai/weknora-ui:latest frontend/

# Build all Docker images
docker-build-all: docker-build-app docker-build-docreader docker-build-frontend

# Run Docker container (traditional way)
docker-run:
	docker-compose up

# Use the new script to start all services
start-all:
	./scripts/start_all.sh

# Use the new script to start only Ollama services
start-ollama:
	./scripts/start_all.sh --ollama

# Use the new script to start only Docker containers
start-docker:
	./scripts/start_all.sh --docker

# Use the new script to stop all services
stop-all:
	./scripts/start_all.sh --stop

# Stop Docker container (traditional way)
docker-stop:
	docker-compose down

# Commands for building images from source
build-images:
	./scripts/build_images.sh

build-images-app:
	./scripts/build_images.sh --app

build-images-docreader:
	./scripts/build_images.sh --docreader

build-images-frontend:
	./scripts/build_images.sh --frontend

clean-images:
	./scripts/build_images.sh --clean

# Restart Docker container (stop, start)
docker-restart:
	docker-compose stop -t 60
	docker-compose up

# Database migrations
migrate-up:
	./scripts/migrate.sh up

migrate-down:
	./scripts/migrate.sh down

migrate-version:
	./scripts/migrate.sh version

migrate-create:
	@if [ -z "$(name)" ]; then \
		echo "Error: migration name is required"; \
		echo "Usage: make migrate-create name=your_migration_name"; \
		exit 1; \
	fi
	./scripts/migrate.sh create $(name)

migrate-force:
	@if [ -z "$(version)" ]; then \
		echo "Error: version is required"; \
		echo "Usage: make migrate-force version=4"; \
		exit 1; \
	fi
	./scripts/migrate.sh force $(version)

migrate-goto:
	@if [ -z "$(version)" ]; then \
		echo "Error: version is required"; \
		echo "Usage: make migrate-goto version=3"; \
		exit 1; \
	fi
	./scripts/migrate.sh goto $(version)

# Generate API documentation (Swagger)
docs:
	@echo "Generating Swagger API documentation..."
	swag init -g $(MAIN_PATH)/main.go -o ./docs --parseDependency --parseInternal
	@echo "Documentation generated under ./docs"
	@echo "Visit http://localhost:8080/swagger/index.html after starting the service to view the docs"

# Install swagger tool
install-swagger:
	go install github.com/swaggo/swag/cmd/swag@latest

# Format code
fmt:
	go fmt ./...

# Lint code
lint:
	golangci-lint run

# Install dependencies
deps:
	go mod download

# Build for production
build-prod:
	VERSION=$${VERSION:-unknown}; \
	COMMIT_ID=$${COMMIT_ID:-unknown}; \
	BUILD_TIME=$${BUILD_TIME:-unknown}; \
	GO_VERSION=$${GO_VERSION:-unknown}; \
	LDFLAGS="-X 'github.com/Tencent/WeKnora/internal/handler.Version=$$VERSION' -X 'github.com/Tencent/WeKnora/internal/handler.CommitID=$$COMMIT_ID' -X 'github.com/Tencent/WeKnora/internal/handler.BuildTime=$$BUILD_TIME' -X 'github.com/Tencent/WeKnora/internal/handler.GoVersion=$$GO_VERSION'"; \
	go build -ldflags="-w -s $$LDFLAGS" -o $(BINARY_NAME) $(MAIN_PATH)

clean-db:
	@echo "Cleaning database..."
	@if [ $$(docker volume ls -q -f name=weknora_postgres-data) ]; then \
		docker volume rm weknora_postgres-data; \
	fi
	@if [ $$(docker volume ls -q -f name=weknora_minio_data) ]; then \
		docker volume rm weknora_minio_data; \
	fi
	@if [ $$(docker volume ls -q -f name=weknora_redis_data) ]; then \
		docker volume rm weknora_redis_data; \
	fi

# Environment check
check-env:
	./scripts/start_all.sh --check

# List containers
list-containers:
	./scripts/start_all.sh --list

# Pull latest images
pull-images:
	./scripts/start_all.sh --pull

# Show current platform
show-platform:
	@echo "Current system architecture: $(shell uname -m)"
	@echo "Docker build platform: $(PLATFORM)"

# Development mode commands
dev-start:
	./scripts/dev.sh start

dev-stop:
	./scripts/dev.sh stop

dev-restart:
	./scripts/dev.sh restart

dev-logs:
	./scripts/dev.sh logs

dev-status:
	./scripts/dev.sh status

dev-app:
	./scripts/dev.sh app

dev-frontend:
	./scripts/dev.sh frontend
