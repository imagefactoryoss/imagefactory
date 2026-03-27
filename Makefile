# Image Factory Makefile

# Variables
BACKEND_DIR := backend
FRONTEND_DIR := frontend
CONTAINER_ENGINE ?= podman
COMPOSE_CMD ?= podman compose
DOCKER_COMPOSE := $(COMPOSE_CMD)
DOCKER_COMPOSE_DEV := $(COMPOSE_CMD) -f docker-compose.yml -f docker-compose.dev.yml
RUNTIME_SERVICE_IMAGES := postgres:15-alpine redis:7-alpine nats:2.10-alpine minio/minio:latest registry:2 axllent/mailpit:latest glauth/glauth:latest
PLATFORMS ?= linux/amd64,linux/arm64
IMAGE_VERSION ?= v0.2.2
IMAGE_ID ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo local)
IMAGE_TAG ?= $(IMAGE_VERSION)-$(IMAGE_ID)
IMAGE_REGISTRY ?=
FRONTEND_USE_LOCAL_DIST ?= true
HELM_RELEASE ?= image-factory
HELM_NAMESPACE ?= image-factory
HELM_CHART ?= ./deploy/helm/image-factory
RELEASE_DIST_DIR ?= ./release/dist
RELEASE_TARGETS ?= linux/amd64 linux/arm64 darwin/amd64 darwin/arm64
SOURCE_IMAGE ?= image-factory-source
SOURCE_EXTRACT_DIR ?= ./dist/image-factory-source
SOURCE_BUILD_CONTEXT ?= ./dist/source-context

# Colors for output
RED := \033[0;31m
GREEN := \033[0;32m
YELLOW := \033[0;33m
BLUE := \033[0;34m
NC := \033[0m # No Color

.PHONY: help
help: ## Show this help message
	@echo "$(BLUE)Image Factory Development Commands$(NC)"
	@echo ""
	@awk 'BEGIN {FS = ":.*##"; printf "Usage:\n  make $(BLUE)<target>$(NC)\n\nTargets:\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  $(BLUE)%-15s$(NC) %s\n", $$1, $$2 } /^##@/ { printf "\n$(YELLOW)%s$(NC)\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Development

.PHONY: dev
dev: ## Start development environment
	@echo "$(GREEN)Starting development environment...$(NC)"
	$(DOCKER_COMPOSE_DEV) up -d
	@echo "$(GREEN)Development environment started!$(NC)"
	@echo "$(YELLOW)Frontend: http://localhost:3000$(NC)"
	@echo "$(YELLOW)Backend: http://localhost:8080$(NC)"
	@echo "$(YELLOW)Adminer: http://localhost:8081$(NC)"
	@echo "$(YELLOW)Mailpit: http://localhost:8025$(NC)"

.PHONY: dev-logs
dev-logs: ## Show development logs
	$(DOCKER_COMPOSE_DEV) logs -f

.PHONY: dev-stop
dev-stop: ## Stop development environment
	@echo "$(YELLOW)Stopping development environment...$(NC)"
	$(DOCKER_COMPOSE_DEV) down

.PHONY: dev-restart
dev-restart: dev-stop dev ## Restart development environment

.PHONY: dev-clean
dev-clean: ## Clean development environment (remove volumes)
	@echo "$(RED)Cleaning development environment...$(NC)"
	$(DOCKER_COMPOSE_DEV) down -v --remove-orphans
	$(CONTAINER_ENGINE) system prune -f

##@ Backend

.PHONY: backend-build
backend-build: ## Build backend
	@echo "$(GREEN)Building backend...$(NC)"
	cd $(BACKEND_DIR) && \
	BUILD_VERSION=$$(git describe --tags --always 2>/dev/null || echo dev) && \
	BUILD_COMMIT=$$(git rev-parse HEAD 2>/dev/null || echo unknown) && \
	BUILD_TIME=$$(date -u +%Y-%m-%dT%H:%M:%SZ) && \
	BUILD_DIRTY=$$(if [ -n "$$(git status --porcelain 2>/dev/null)" ]; then echo true; else echo false; fi) && \
	go build -ldflags "-X main.buildVersion=$$BUILD_VERSION -X main.buildCommit=$$BUILD_COMMIT -X main.buildTime=$$BUILD_TIME -X main.buildDirty=$$BUILD_DIRTY" -o server ./cmd/server

.PHONY: backend-build-email-worker
backend-build-email-worker: ## Build email worker
	@echo "$(GREEN)Building email worker...$(NC)"
	cd $(BACKEND_DIR) && \
	BUILD_VERSION=$$(git describe --tags --always 2>/dev/null || echo dev) && \
	BUILD_COMMIT=$$(git rev-parse HEAD 2>/dev/null || echo unknown) && \
	BUILD_TIME=$$(date -u +%Y-%m-%dT%H:%M:%SZ) && \
	BUILD_DIRTY=$$(if [ -n "$$(git status --porcelain 2>/dev/null)" ]; then echo true; else echo false; fi) && \
	go build -ldflags "-X main.buildVersion=$$BUILD_VERSION -X main.buildCommit=$$BUILD_COMMIT -X main.buildTime=$$BUILD_TIME -X main.buildDirty=$$BUILD_DIRTY" -o email-worker ./cmd/email-worker

.PHONY: backend-test
backend-test: ## Run backend tests
	@echo "$(GREEN)Running backend tests...$(NC)"
	cd $(BACKEND_DIR) && go test ./...

.PHONY: backend-test-coverage
backend-test-coverage: ## Run backend tests with coverage
	@echo "$(GREEN)Running backend tests with coverage...$(NC)"
	cd $(BACKEND_DIR) && go test -coverprofile=coverage.out ./...
	cd $(BACKEND_DIR) && go tool cover -html=coverage.out -o coverage.html

.PHONY: backend-coverage-analysis
backend-coverage-analysis: ## Run detailed coverage analysis
	@echo "$(GREEN)Running detailed coverage analysis...$(NC)"
	@bash scripts/coverage_analysis.sh

.PHONY: backend-lint
backend-lint: ## Lint backend code
	@echo "$(GREEN)Linting backend code...$(NC)"
	cd $(BACKEND_DIR) && golangci-lint run

.PHONY: backend-format
backend-format: ## Format backend code
	@echo "$(GREEN)Formatting backend code...$(NC)"
	cd $(BACKEND_DIR) && go fmt ./...

.PHONY: backend-mod-tidy
backend-mod-tidy: ## Tidy backend modules
	@echo "$(GREEN)Tidying backend modules...$(NC)"
	cd $(BACKEND_DIR) && go mod tidy

.PHONY: backend-migrate-up
backend-migrate-up: ## Run database migrations up
	@echo "$(GREEN)Running database migrations up...$(NC)"
	cd $(BACKEND_DIR) && go run cmd/migrate/main.go up

.PHONY: backend-migrate-down
backend-migrate-down: ## Run database migrations down
	@echo "$(YELLOW)Running database migrations down...$(NC)"
	cd $(BACKEND_DIR) && go run cmd/migrate/main.go down

.PHONY: backend-seed
backend-seed: ## Seed database with test data
	@echo "$(GREEN)Seeding database...$(NC)"
	cd $(BACKEND_DIR) && go run cmd/seed/main.go

##@ Frontend

.PHONY: frontend-install
frontend-install: ## Install frontend dependencies
	@echo "$(GREEN)Installing frontend dependencies...$(NC)"
	cd $(FRONTEND_DIR) && npm install

.PHONY: frontend-build
frontend-build: ## Build frontend
	@echo "$(GREEN)Building frontend...$(NC)"
	cd $(FRONTEND_DIR) && npm run build

.PHONY: frontend-test
frontend-test: ## Run frontend tests
	@echo "$(GREEN)Running frontend tests...$(NC)"
	cd $(FRONTEND_DIR) && npm run test

.PHONY: frontend-test-coverage
frontend-test-coverage: ## Run frontend tests with coverage
	@echo "$(GREEN)Running frontend tests with coverage...$(NC)"
	cd $(FRONTEND_DIR) && npm run test:coverage

.PHONY: frontend-lint
frontend-lint: ## Lint frontend code
	@echo "$(GREEN)Linting frontend code...$(NC)"
	cd $(FRONTEND_DIR) && npm run lint

.PHONY: frontend-lint-fix
frontend-lint-fix: ## Fix frontend lint errors
	@echo "$(GREEN)Fixing frontend lint errors...$(NC)"
	cd $(FRONTEND_DIR) && npm run lint:fix

.PHONY: frontend-format
frontend-format: ## Format frontend code
	@echo "$(GREEN)Formatting frontend code...$(NC)"
	cd $(FRONTEND_DIR) && npm run format

.PHONY: frontend-type-check
frontend-type-check: ## Type check frontend code
	@echo "$(GREEN)Type checking frontend code...$(NC)"
	cd $(FRONTEND_DIR) && npm run type-check

##@ Production

.PHONY: prod
prod: ## Start production environment
	@echo "$(GREEN)Starting production environment...$(NC)"
	$(DOCKER_COMPOSE) up -d
	@echo "$(GREEN)Production environment started!$(NC)"

.PHONY: prod-build
prod-build: ## Build production images
	@echo "$(GREEN)Building production images...$(NC)"
	$(DOCKER_COMPOSE) build

.PHONY: prod-stop
prod-stop: ## Stop production environment
	@echo "$(YELLOW)Stopping production environment...$(NC)"
	$(DOCKER_COMPOSE) down

.PHONY: prod-logs
prod-logs: ## Show production logs
	$(DOCKER_COMPOSE) logs -f

##@ Database

.PHONY: db-shell
db-shell: ## Connect to database shell
	@echo "$(GREEN)Connecting to database...$(NC)"
	$(CONTAINER_ENGINE) exec -it image-factory-postgres psql -U postgres -d image_factory_dev

.PHONY: db-backup
db-backup: ## Backup database
	@echo "$(GREEN)Backing up database...$(NC)"
	$(CONTAINER_ENGINE) exec image-factory-postgres pg_dump -U postgres image_factory_dev > backup_$$(date +%Y%m%d_%H%M%S).sql

.PHONY: db-restore
db-restore: ## Restore database from backup (usage: make db-restore BACKUP=backup_file.sql)
	@echo "$(GREEN)Restoring database from $(BACKUP)...$(NC)"
	$(CONTAINER_ENGINE) exec -i image-factory-postgres psql -U postgres image_factory_dev < $(BACKUP)

##@ Testing

.PHONY: test
test: backend-test frontend-test ## Run all tests

.PHONY: test-coverage
test-coverage: backend-test-coverage frontend-test-coverage ## Run all tests with coverage

.PHONY: test-integration
test-integration: ## Run integration tests
	@echo "$(GREEN)Running integration tests...$(NC)"
	cd $(BACKEND_DIR) && go test -tags=integration ./tests/integration/...

.PHONY: test-e2e
test-e2e: ## Run end-to-end tests
	@echo "$(GREEN)Running end-to-end tests...$(NC)"
	cd $(FRONTEND_DIR) && npm run test:e2e

##@ Quality

.PHONY: lint
lint: backend-lint frontend-lint ## Lint all code

.PHONY: format
format: backend-format frontend-format ## Format all code

.PHONY: quality-check
quality-check: lint test ## Run quality checks

##@ Release

.PHONY: release-binaries
release-binaries: ## Build release tarballs for runtime binaries
	@echo "$(GREEN)Building release binaries for $(RELEASE_TARGETS)...$(NC)"
	VERSION=$(IMAGE_VERSION) TARGETS="$(RELEASE_TARGETS)" DIST_DIR=$(RELEASE_DIST_DIR) ./scripts/release-binaries.sh

.PHONY: release-upload-assets
release-upload-assets: ## Upload built release tarballs to GitHub release (requires TAG)
	@echo "$(GREEN)Uploading release assets for $(TAG)...$(NC)"
	@test -n "$(TAG)" || { echo "$(RED)TAG is required. Example: make release-upload-assets TAG=v0.1.0$(NC)"; exit 1; }
	TAG=$(TAG) DIST_DIR=$(RELEASE_DIST_DIR) ./scripts/upload-release-assets.sh

##@ Docker

.PHONY: docker-build
docker-build: ## Build Docker images
	@echo "$(GREEN)Building container images with $(CONTAINER_ENGINE)...$(NC)"
	@FRONTEND_TARGET=runtime; \
	if [ "$(FRONTEND_USE_LOCAL_DIST)" = "true" ]; then \
		echo "$(YELLOW)Using local frontend dist for image build...$(NC)"; \
		(cd $(FRONTEND_DIR) && (npm run build || npx vite build)); \
		if [ ! -d "$(FRONTEND_DIR)/dist" ]; then \
			echo "$(RED)frontend/dist not found after local build. Aborting.$(NC)"; \
			exit 1; \
		fi; \
		FRONTEND_TARGET=runtime-local; \
	else \
		echo "$(YELLOW)Using container frontend build stage...$(NC)"; \
	fi; \
	$(CONTAINER_ENGINE) build -t image-factory-backend:latest $(BACKEND_DIR); \
	$(CONTAINER_ENGINE) build --target $$FRONTEND_TARGET -t image-factory-frontend:latest $(FRONTEND_DIR); \
	$(CONTAINER_ENGINE) build -f Dockerfile.docs -t image-factory-docs:latest .

.PHONY: docker-build-workers
docker-build-workers: ## Build worker Docker images
	@echo "$(GREEN)Building worker images with $(CONTAINER_ENGINE)...$(NC)"
	$(CONTAINER_ENGINE) build -f $(BACKEND_DIR)/Dockerfile.dispatcher -t image-factory-dispatcher:latest $(BACKEND_DIR)
	$(CONTAINER_ENGINE) build -f $(BACKEND_DIR)/Dockerfile.notification-worker -t image-factory-notification-worker:latest $(BACKEND_DIR)
	$(CONTAINER_ENGINE) build -f $(BACKEND_DIR)/Dockerfile.email-worker -t image-factory-email-worker:latest $(BACKEND_DIR)
	$(CONTAINER_ENGINE) build -f $(BACKEND_DIR)/Dockerfile.internal-registry-gc-worker -t image-factory-internal-registry-gc-worker:latest $(BACKEND_DIR)

.PHONY: docker-build-source
docker-build-source: ## Build source distribution image from tracked files at HEAD
	@echo "$(GREEN)Building source distribution image with $(CONTAINER_ENGINE)...$(NC)"
	@mkdir -p $(dir $(SOURCE_BUILD_CONTEXT))
	@rm -rf $(SOURCE_BUILD_CONTEXT)
	@mkdir -p $(SOURCE_BUILD_CONTEXT)
	@git archive --format=tar HEAD | tar -xf - -C $(SOURCE_BUILD_CONTEXT)
	$(CONTAINER_ENGINE) build -f Dockerfile.source -t $(SOURCE_IMAGE):$(IMAGE_TAG) $(SOURCE_BUILD_CONTEXT)

.PHONY: docker-extract-source
docker-extract-source: ## Extract source tree from source distribution image
	@echo "$(GREEN)Extracting source from $(SOURCE_IMAGE):$(IMAGE_TAG) to $(SOURCE_EXTRACT_DIR)...$(NC)"
	@rm -rf $(SOURCE_EXTRACT_DIR)
	@mkdir -p $(SOURCE_EXTRACT_DIR)
	@CONTAINER_ID=$$($(CONTAINER_ENGINE) create $(SOURCE_IMAGE):$(IMAGE_TAG)); \
		trap "$(CONTAINER_ENGINE) rm -f $$CONTAINER_ID >/dev/null 2>/dev/null || true" EXIT; \
		$(CONTAINER_ENGINE) cp $$CONTAINER_ID:/src/. $(SOURCE_EXTRACT_DIR); \
		$(CONTAINER_ENGINE) rm $$CONTAINER_ID >/dev/null; \
		trap - EXIT

.PHONY: build-all-images docker-build-all
build-all-images: docker-build docker-build-workers ## Build app and worker container images
docker-build-all: build-all-images

.PHONY: docker-build-all-multiarch
docker-build-all-multiarch: ## Build amd64/arm64 images and push manifest list (requires IMAGE_REGISTRY)
	@echo "$(GREEN)Building multi-platform images for $(PLATFORMS) using $(CONTAINER_ENGINE)...$(NC)"
	@test -n "$(IMAGE_REGISTRY)" || { echo "$(RED)IMAGE_REGISTRY is required. Example: make docker-build-all-multiarch IMAGE_REGISTRY=ghcr.io/acme$(NC)"; exit 1; }
	@FRONTEND_TARGET=runtime; \
	if [ "$(FRONTEND_USE_LOCAL_DIST)" = "true" ]; then \
		echo "$(YELLOW)Using local frontend dist for image build...$(NC)"; \
		(cd $(FRONTEND_DIR) && (npm run build || npx vite build)); \
		if [ ! -d "$(FRONTEND_DIR)/dist" ]; then \
			echo "$(RED)frontend/dist not found after local build. Aborting.$(NC)"; \
			exit 1; \
		fi; \
		FRONTEND_TARGET=runtime-local; \
	else \
		echo "$(YELLOW)Using container frontend build stage...$(NC)"; \
	fi; \
	if [ "$(CONTAINER_ENGINE)" = "docker" ]; then \
		docker buildx build --platform $(PLATFORMS) -t $(IMAGE_REGISTRY)/image-factory-backend:$(IMAGE_TAG) --push $(BACKEND_DIR); \
		docker buildx build --target $$FRONTEND_TARGET --platform $(PLATFORMS) -t $(IMAGE_REGISTRY)/image-factory-frontend:$(IMAGE_TAG) --push $(FRONTEND_DIR); \
		docker buildx build --platform $(PLATFORMS) -f Dockerfile.docs -t $(IMAGE_REGISTRY)/image-factory-docs:$(IMAGE_TAG) --push .; \
		docker buildx build --platform $(PLATFORMS) -f $(BACKEND_DIR)/Dockerfile.dispatcher -t $(IMAGE_REGISTRY)/image-factory-dispatcher:$(IMAGE_TAG) --push $(BACKEND_DIR); \
		docker buildx build --platform $(PLATFORMS) -f $(BACKEND_DIR)/Dockerfile.notification-worker -t $(IMAGE_REGISTRY)/image-factory-notification-worker:$(IMAGE_TAG) --push $(BACKEND_DIR); \
		docker buildx build --platform $(PLATFORMS) -f $(BACKEND_DIR)/Dockerfile.email-worker -t $(IMAGE_REGISTRY)/image-factory-email-worker:$(IMAGE_TAG) --push $(BACKEND_DIR); \
		docker buildx build --platform $(PLATFORMS) -f $(BACKEND_DIR)/Dockerfile.internal-registry-gc-worker -t $(IMAGE_REGISTRY)/image-factory-internal-registry-gc-worker:$(IMAGE_TAG) --push $(BACKEND_DIR); \
	elif [ "$(CONTAINER_ENGINE)" = "podman" ]; then \
		podman build --platform $(PLATFORMS) --manifest $(IMAGE_REGISTRY)/image-factory-backend:$(IMAGE_TAG) $(BACKEND_DIR); \
		podman manifest push --all $(IMAGE_REGISTRY)/image-factory-backend:$(IMAGE_TAG) docker://$(IMAGE_REGISTRY)/image-factory-backend:$(IMAGE_TAG); \
		podman build --target $$FRONTEND_TARGET --platform $(PLATFORMS) --manifest $(IMAGE_REGISTRY)/image-factory-frontend:$(IMAGE_TAG) $(FRONTEND_DIR); \
		podman manifest push --all $(IMAGE_REGISTRY)/image-factory-frontend:$(IMAGE_TAG) docker://$(IMAGE_REGISTRY)/image-factory-frontend:$(IMAGE_TAG); \
		podman build --platform $(PLATFORMS) -f Dockerfile.docs --manifest $(IMAGE_REGISTRY)/image-factory-docs:$(IMAGE_TAG) .; \
		podman manifest push --all $(IMAGE_REGISTRY)/image-factory-docs:$(IMAGE_TAG) docker://$(IMAGE_REGISTRY)/image-factory-docs:$(IMAGE_TAG); \
		podman build --platform $(PLATFORMS) -f $(BACKEND_DIR)/Dockerfile.dispatcher --manifest $(IMAGE_REGISTRY)/image-factory-dispatcher:$(IMAGE_TAG) $(BACKEND_DIR); \
		podman manifest push --all $(IMAGE_REGISTRY)/image-factory-dispatcher:$(IMAGE_TAG) docker://$(IMAGE_REGISTRY)/image-factory-dispatcher:$(IMAGE_TAG); \
		podman build --platform $(PLATFORMS) -f $(BACKEND_DIR)/Dockerfile.notification-worker --manifest $(IMAGE_REGISTRY)/image-factory-notification-worker:$(IMAGE_TAG) $(BACKEND_DIR); \
		podman manifest push --all $(IMAGE_REGISTRY)/image-factory-notification-worker:$(IMAGE_TAG) docker://$(IMAGE_REGISTRY)/image-factory-notification-worker:$(IMAGE_TAG); \
		podman build --platform $(PLATFORMS) -f $(BACKEND_DIR)/Dockerfile.email-worker --manifest $(IMAGE_REGISTRY)/image-factory-email-worker:$(IMAGE_TAG) $(BACKEND_DIR); \
		podman manifest push --all $(IMAGE_REGISTRY)/image-factory-email-worker:$(IMAGE_TAG) docker://$(IMAGE_REGISTRY)/image-factory-email-worker:$(IMAGE_TAG); \
		podman build --platform $(PLATFORMS) -f $(BACKEND_DIR)/Dockerfile.internal-registry-gc-worker --manifest $(IMAGE_REGISTRY)/image-factory-internal-registry-gc-worker:$(IMAGE_TAG) $(BACKEND_DIR); \
		podman manifest push --all $(IMAGE_REGISTRY)/image-factory-internal-registry-gc-worker:$(IMAGE_TAG) docker://$(IMAGE_REGISTRY)/image-factory-internal-registry-gc-worker:$(IMAGE_TAG); \
	else \
		echo "$(RED)Unsupported CONTAINER_ENGINE=$(CONTAINER_ENGINE). Use docker or podman.$(NC)"; \
		exit 1; \
	fi

.PHONY: release-deploy
release-deploy: ## Build+push multiarch images and helm upgrade release with IMAGE_TAG
	@echo "$(GREEN)Release deploy with tag $(IMAGE_TAG)$(NC)"
	@test -n "$(IMAGE_REGISTRY)" || { echo "$(RED)IMAGE_REGISTRY is required. Example: make release-deploy IMAGE_REGISTRY=registry.gitlab.com/imagefactoryoss/imagefactory$(NC)"; exit 1; }
	@$(MAKE) docker-build-all-multiarch IMAGE_REGISTRY=$(IMAGE_REGISTRY) IMAGE_TAG=$(IMAGE_TAG) CONTAINER_ENGINE=$(CONTAINER_ENGINE) PLATFORMS=$(PLATFORMS) FRONTEND_USE_LOCAL_DIST=$(FRONTEND_USE_LOCAL_DIST)
	helm upgrade --install $(HELM_RELEASE) $(HELM_CHART) -n $(HELM_NAMESPACE) --reuse-values \
		--set backend.image.repository=$(IMAGE_REGISTRY)/image-factory-backend \
		--set backend.image.tag=$(IMAGE_TAG) \
		--set backend.image.pullPolicy=Always \
		--set frontend.image.repository=$(IMAGE_REGISTRY)/image-factory-frontend \
		--set frontend.image.tag=$(IMAGE_TAG) \
		--set frontend.image.pullPolicy=Always \
		--set docs.image.repository=$(IMAGE_REGISTRY)/image-factory-docs \
		--set docs.image.tag=$(IMAGE_TAG) \
		--set docs.image.pullPolicy=Always \
		--set workers.dispatcher.image.repository=$(IMAGE_REGISTRY)/image-factory-dispatcher \
		--set workers.dispatcher.image.tag=$(IMAGE_TAG) \
		--set workers.notification.image.repository=$(IMAGE_REGISTRY)/image-factory-notification-worker \
		--set workers.notification.image.tag=$(IMAGE_TAG) \
		--set workers.email.image.repository=$(IMAGE_REGISTRY)/image-factory-email-worker \
		--set workers.email.image.tag=$(IMAGE_TAG) \
		--set workers.internalRegistryGc.image.repository=$(IMAGE_REGISTRY)/image-factory-internal-registry-gc-worker \
		--set workers.internalRegistryGc.image.tag=$(IMAGE_TAG)
	kubectl -n $(HELM_NAMESPACE) rollout status deployment/$(HELM_RELEASE)-backend --timeout=300s
	kubectl -n $(HELM_NAMESPACE) rollout status deployment/$(HELM_RELEASE)-frontend --timeout=300s
	kubectl -n $(HELM_NAMESPACE) rollout status deployment/$(HELM_RELEASE)-docs --timeout=300s

.PHONY: docker-pull-runtime
docker-pull-runtime: ## Pre-pull runtime service images used by Helm chart
	@echo "$(GREEN)Pulling runtime dependency images with $(CONTAINER_ENGINE)...$(NC)"
	@for image in $(RUNTIME_SERVICE_IMAGES); do \
		echo "Pulling $$image"; \
		$(CONTAINER_ENGINE) pull $$image; \
	done

.PHONY: docker-push
docker-push: ## Push Docker images
	@echo "$(GREEN)Pushing container images with $(CONTAINER_ENGINE)...$(NC)"
	# Add your registry URL here
	# docker tag image-factory-backend:latest your-registry/image-factory-backend:latest
	# docker push your-registry/image-factory-backend:latest

.PHONY: docker-clean
docker-clean: ## Clean Docker resources
	@echo "$(YELLOW)Cleaning container resources with $(CONTAINER_ENGINE)...$(NC)"
	$(CONTAINER_ENGINE) system prune -af
	$(CONTAINER_ENGINE) volume prune -f

##@ Monitoring

.PHONY: logs
logs: ## Show all logs
	$(DOCKER_COMPOSE) logs -f

.PHONY: stats
stats: ## Show Docker stats
	$(CONTAINER_ENGINE) stats

.PHONY: ps
ps: ## Show running containers
	$(DOCKER_COMPOSE) ps

##@ Utilities

.PHONY: install
install: frontend-install backend-mod-tidy ## Install all dependencies

.PHONY: clean
clean: dev-clean docker-clean ## Clean everything

.PHONY: reset
reset: clean install ## Reset entire project

.PHONY: check-tools
check-tools: ## Check if required tools are installed
	@echo "$(GREEN)Checking required tools...$(NC)"
	@command -v go >/dev/null 2>&1 || { echo "$(RED)Go is not installed$(NC)"; exit 1; }
	@command -v node >/dev/null 2>&1 || { echo "$(RED)Node.js is not installed$(NC)"; exit 1; }
	@command -v $(CONTAINER_ENGINE) >/dev/null 2>&1 || { echo "$(RED)$(CONTAINER_ENGINE) is not installed$(NC)"; exit 1; }
	@command -v $(word 1,$(COMPOSE_CMD)) >/dev/null 2>&1 || { echo "$(RED)$(word 1,$(COMPOSE_CMD)) is not installed$(NC)"; exit 1; }
	@echo "$(GREEN)All required tools are installed!$(NC)"

.PHONY: setup
setup: check-tools install ## Initial project setup

##@ CI/CD

.PHONY: ci
ci: quality-check test-coverage ## Run CI pipeline

.PHONY: cd
cd: docker-build docker-push ## Run CD pipeline

# Default target
.DEFAULT_GOAL := help
