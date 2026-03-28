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
IMAGE_VERSION ?= v0.3.0
IMAGE_ID ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo local)
IMAGE_TAG ?= $(IMAGE_VERSION)-$(IMAGE_ID)
IMAGE_REGISTRY ?=
APP_VERSION ?= $(patsubst v%,%,$(IMAGE_VERSION))
APP_BUILD_DATE ?= $(shell date -u +%Y-%m-%d)
PRODUCT_INFO_LAST_SYNC ?= $(APP_BUILD_DATE)
FRONTEND_USE_LOCAL_DIST ?= true
HELM_RELEASE ?= image-factory
HELM_NAMESPACE ?= image-factory
HELM_CHART ?= ./deploy/helm/image-factory
OLLAMA_MODEL_STORE ?= $(HOME)/.ollama
OLLAMA_MODEL_NAME ?= llama3.2:3b
OLLAMA_IMAGE_TAG ?= image-factory-ollama:llama3.2-3b
OLLAMA_BASE_IMAGE ?= docker.io/ollama/ollama:latest
OLLAMA_ENABLED ?= false
OLLAMA_STORAGE_MODE ?= baked
OLLAMA_IMAGE_REPOSITORY ?= $(IMAGE_REGISTRY)/image-factory-ollama

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

.PHONY: qa-packer-provider-native-matrix
qa-packer-provider-native-matrix: ## Run Packer provider-native matrix validation (defaults to mock mode; writes docs/qa/artifacts log)
	@./scripts/qa/packer_provider_native_matrix_validate.sh

.PHONY: qa-sre-smartbot-regression
qa-sre-smartbot-regression: ## Run SRE Smart Bot focused backend+frontend regression suite (writes docs/qa/artifacts log)
	@./scripts/qa/sre_smartbot_regression_validate.sh

##@ Quality

.PHONY: lint
lint: backend-lint frontend-lint ## Lint all code

.PHONY: format
format: backend-format frontend-format ## Format all code

.PHONY: quality-check
quality-check: lint test ## Run quality checks

##@ Docker

.PHONY: docker-build
docker-build: ## Build Docker images
	@echo "$(GREEN)Building container images with $(CONTAINER_ENGINE)...$(NC)"
	@FRONTEND_TARGET=runtime; \
	if [ "$(FRONTEND_USE_LOCAL_DIST)" = "true" ]; then \
		echo "$(YELLOW)Using local frontend dist for image build...$(NC)"; \
		(cd $(FRONTEND_DIR) && (VITE_APP_VERSION=$(APP_VERSION) VITE_APP_BUILD_DATE=$(APP_BUILD_DATE) VITE_PRODUCT_INFO_LAST_SYNC=$(PRODUCT_INFO_LAST_SYNC) npm run build || VITE_APP_VERSION=$(APP_VERSION) VITE_APP_BUILD_DATE=$(APP_BUILD_DATE) VITE_PRODUCT_INFO_LAST_SYNC=$(PRODUCT_INFO_LAST_SYNC) npx vite build)); \
		if [ ! -d "$(FRONTEND_DIR)/dist" ]; then \
			echo "$(RED)frontend/dist not found after local build. Aborting.$(NC)"; \
			exit 1; \
		fi; \
		FRONTEND_TARGET=runtime-local; \
	else \
		echo "$(YELLOW)Using container frontend build stage...$(NC)"; \
	fi; \
	$(CONTAINER_ENGINE) build -t image-factory-backend:latest $(BACKEND_DIR); \
	$(CONTAINER_ENGINE) build --build-arg VITE_APP_VERSION=$(APP_VERSION) --build-arg VITE_APP_BUILD_DATE=$(APP_BUILD_DATE) --build-arg VITE_PRODUCT_INFO_LAST_SYNC=$(PRODUCT_INFO_LAST_SYNC) --target $$FRONTEND_TARGET -t image-factory-frontend:latest $(FRONTEND_DIR); \
	$(CONTAINER_ENGINE) build -f Dockerfile.docs -t image-factory-docs:latest .

.PHONY: docker-build-workers
docker-build-workers: ## Build worker Docker images
	@echo "$(GREEN)Building worker images with $(CONTAINER_ENGINE)...$(NC)"
	$(CONTAINER_ENGINE) build -f $(BACKEND_DIR)/Dockerfile.dispatcher -t image-factory-dispatcher:latest $(BACKEND_DIR)
	$(CONTAINER_ENGINE) build -f $(BACKEND_DIR)/Dockerfile.notification-worker -t image-factory-notification-worker:latest $(BACKEND_DIR)
	$(CONTAINER_ENGINE) build -f $(BACKEND_DIR)/Dockerfile.email-worker -t image-factory-email-worker:latest $(BACKEND_DIR)
	$(CONTAINER_ENGINE) build -f $(BACKEND_DIR)/Dockerfile.internal-registry-gc-worker -t image-factory-internal-registry-gc-worker:latest $(BACKEND_DIR)

.PHONY: docker-export-source-tar
docker-export-source-tar: ## Export source from image to tar (set IMAGE=<image>, optional SERVE=true PORT=8089)
	@test -n "$(IMAGE)" || { echo "$(RED)IMAGE is required. Example: make docker-export-source-tar IMAGE=image-factory-backend:latest$(NC)"; exit 1; }
	@ARGS="--image $(IMAGE) --engine $(CONTAINER_ENGINE)"; \
	if [ -n "$(SOURCE_PATH)" ]; then ARGS="$$ARGS --source-path $(SOURCE_PATH)"; fi; \
	if [ -n "$(OUTPUT_DIR)" ]; then ARGS="$$ARGS --output-dir $(OUTPUT_DIR)"; fi; \
	if [ -n "$(TAR_NAME)" ]; then ARGS="$$ARGS --tar-name $(TAR_NAME)"; fi; \
	if [ "$(SERVE)" = "true" ]; then ARGS="$$ARGS --serve"; fi; \
	if [ -n "$(PORT)" ]; then ARGS="$$ARGS --port $(PORT)"; fi; \
	if [ -n "$(BIND_ADDR)" ]; then ARGS="$$ARGS --bind $(BIND_ADDR)"; fi; \
	./scripts/export-image-source-tar.sh $$ARGS

.PHONY: docker-build-ollama-baked
docker-build-ollama-baked: ## Build a baked Ollama image from a pre-seeded local model store
	@echo "$(GREEN)Building baked Ollama image with $(CONTAINER_ENGINE)...$(NC)"
	./scripts/build-baked-ollama-image.sh \
		--engine "$(CONTAINER_ENGINE)" \
		--source "$(OLLAMA_MODEL_STORE)" \
		--tag "$(OLLAMA_IMAGE_TAG)" \
		--base-image "$(OLLAMA_BASE_IMAGE)" \
		--model "$(OLLAMA_MODEL_NAME)"

.PHONY: docker-build-ollama-baked-push
docker-build-ollama-baked-push: ## Build and push a baked Ollama image (single-arch / pre-seeded local store)
	@echo "$(GREEN)Building and pushing baked Ollama image with $(CONTAINER_ENGINE)...$(NC)"
	@test -n "$(IMAGE_REGISTRY)" || { echo "$(RED)IMAGE_REGISTRY is required. Example: make docker-build-ollama-baked-push IMAGE_REGISTRY=registry.gitlab.com/s4cna/image-factory$(NC)"; exit 1; }
	./scripts/build-baked-ollama-image.sh \
		--engine "$(CONTAINER_ENGINE)" \
		--source "$(OLLAMA_MODEL_STORE)" \
		--tag "$(OLLAMA_IMAGE_REPOSITORY):$(IMAGE_TAG)" \
		--base-image "$(OLLAMA_BASE_IMAGE)" \
		--model "$(OLLAMA_MODEL_NAME)"
	$(CONTAINER_ENGINE) push "$(OLLAMA_IMAGE_REPOSITORY):$(IMAGE_TAG)"

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
		(cd $(FRONTEND_DIR) && (VITE_APP_VERSION=$(APP_VERSION) VITE_APP_BUILD_DATE=$(APP_BUILD_DATE) VITE_PRODUCT_INFO_LAST_SYNC=$(PRODUCT_INFO_LAST_SYNC) npm run build || VITE_APP_VERSION=$(APP_VERSION) VITE_APP_BUILD_DATE=$(APP_BUILD_DATE) VITE_PRODUCT_INFO_LAST_SYNC=$(PRODUCT_INFO_LAST_SYNC) npx vite build)); \
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
		docker buildx build --build-arg VITE_APP_VERSION=$(APP_VERSION) --build-arg VITE_APP_BUILD_DATE=$(APP_BUILD_DATE) --build-arg VITE_PRODUCT_INFO_LAST_SYNC=$(PRODUCT_INFO_LAST_SYNC) --target $$FRONTEND_TARGET --platform $(PLATFORMS) -t $(IMAGE_REGISTRY)/image-factory-frontend:$(IMAGE_TAG) --push $(FRONTEND_DIR); \
		docker buildx build --platform $(PLATFORMS) -f Dockerfile.docs -t $(IMAGE_REGISTRY)/image-factory-docs:$(IMAGE_TAG) --push .; \
		docker buildx build --platform $(PLATFORMS) -f $(BACKEND_DIR)/Dockerfile.dispatcher -t $(IMAGE_REGISTRY)/image-factory-dispatcher:$(IMAGE_TAG) --push $(BACKEND_DIR); \
		docker buildx build --platform $(PLATFORMS) -f $(BACKEND_DIR)/Dockerfile.notification-worker -t $(IMAGE_REGISTRY)/image-factory-notification-worker:$(IMAGE_TAG) --push $(BACKEND_DIR); \
		docker buildx build --platform $(PLATFORMS) -f $(BACKEND_DIR)/Dockerfile.email-worker -t $(IMAGE_REGISTRY)/image-factory-email-worker:$(IMAGE_TAG) --push $(BACKEND_DIR); \
		docker buildx build --platform $(PLATFORMS) -f $(BACKEND_DIR)/Dockerfile.internal-registry-gc-worker -t $(IMAGE_REGISTRY)/image-factory-internal-registry-gc-worker:$(IMAGE_TAG) --push $(BACKEND_DIR); \
	elif [ "$(CONTAINER_ENGINE)" = "podman" ]; then \
		podman build --platform $(PLATFORMS) --manifest $(IMAGE_REGISTRY)/image-factory-backend:$(IMAGE_TAG) $(BACKEND_DIR); \
		podman manifest push --all $(IMAGE_REGISTRY)/image-factory-backend:$(IMAGE_TAG) docker://$(IMAGE_REGISTRY)/image-factory-backend:$(IMAGE_TAG); \
		podman build --build-arg VITE_APP_VERSION=$(APP_VERSION) --build-arg VITE_APP_BUILD_DATE=$(APP_BUILD_DATE) --build-arg VITE_PRODUCT_INFO_LAST_SYNC=$(PRODUCT_INFO_LAST_SYNC) --target $$FRONTEND_TARGET --platform $(PLATFORMS) --manifest $(IMAGE_REGISTRY)/image-factory-frontend:$(IMAGE_TAG) $(FRONTEND_DIR); \
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

.PHONY: docker-build-docs-multiarch
docker-build-docs-multiarch: ## Build docs image for PLATFORMS and push (requires IMAGE_REGISTRY)
	@echo "$(GREEN)Building docs image for $(PLATFORMS) using $(CONTAINER_ENGINE)...$(NC)"
	@test -n "$(IMAGE_REGISTRY)" || { echo "$(RED)IMAGE_REGISTRY is required. Example: make docker-build-docs-multiarch IMAGE_REGISTRY=ghcr.io/acme$(NC)"; exit 1; }
	@if [ "$(CONTAINER_ENGINE)" = "docker" ]; then \
		docker buildx build --platform $(PLATFORMS) -f Dockerfile.docs -t $(IMAGE_REGISTRY)/image-factory-docs:$(IMAGE_TAG) --push .; \
	elif [ "$(CONTAINER_ENGINE)" = "podman" ]; then \
		podman build --platform $(PLATFORMS) -f Dockerfile.docs --manifest $(IMAGE_REGISTRY)/image-factory-docs:$(IMAGE_TAG) .; \
		podman manifest push --all $(IMAGE_REGISTRY)/image-factory-docs:$(IMAGE_TAG) docker://$(IMAGE_REGISTRY)/image-factory-docs:$(IMAGE_TAG); \
	else \
		echo "$(RED)Unsupported CONTAINER_ENGINE=$(CONTAINER_ENGINE). Use docker or podman.$(NC)"; \
		exit 1; \
	fi

.PHONY: release-deploy
release-deploy: ## Build+push multiarch images and helm upgrade release with IMAGE_TAG
	@echo "$(GREEN)Release deploy with tag $(IMAGE_TAG)$(NC)"
	@test -n "$(IMAGE_REGISTRY)" || { echo "$(RED)IMAGE_REGISTRY is required. Example: make release-deploy IMAGE_REGISTRY=registry.gitlab.com/s4cna/image-factory$(NC)"; exit 1; }
	@$(MAKE) docker-build-all-multiarch IMAGE_REGISTRY=$(IMAGE_REGISTRY) IMAGE_TAG=$(IMAGE_TAG) CONTAINER_ENGINE=$(CONTAINER_ENGINE) PLATFORMS=$(PLATFORMS) FRONTEND_USE_LOCAL_DIST=$(FRONTEND_USE_LOCAL_DIST)
	@if [ "$(OLLAMA_ENABLED)" = "true" ] && [ "$(OLLAMA_STORAGE_MODE)" = "baked" ]; then \
		$(MAKE) docker-build-ollama-baked-push IMAGE_REGISTRY=$(IMAGE_REGISTRY) IMAGE_TAG=$(IMAGE_TAG) CONTAINER_ENGINE=$(CONTAINER_ENGINE) OLLAMA_MODEL_STORE="$(OLLAMA_MODEL_STORE)" OLLAMA_MODEL_NAME="$(OLLAMA_MODEL_NAME)" OLLAMA_BASE_IMAGE="$(OLLAMA_BASE_IMAGE)" OLLAMA_IMAGE_REPOSITORY="$(OLLAMA_IMAGE_REPOSITORY)"; \
	fi
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
		--set workers.internalRegistryGc.image.tag=$(IMAGE_TAG) \
		--set ollama.enabled=$(OLLAMA_ENABLED) \
		--set ollama.storage.mode=$(OLLAMA_STORAGE_MODE) \
		--set ollama.image.repository=$(OLLAMA_IMAGE_REPOSITORY) \
		--set ollama.image.tag=$(IMAGE_TAG)
	kubectl -n $(HELM_NAMESPACE) rollout status deployment/$(HELM_RELEASE)-backend --timeout=300s
	kubectl -n $(HELM_NAMESPACE) rollout status deployment/$(HELM_RELEASE)-frontend --timeout=300s
	kubectl -n $(HELM_NAMESPACE) rollout status deployment/$(HELM_RELEASE)-docs --timeout=300s

.PHONY: release
release: release-deploy ## Alias for release deploy

.PHONY: release-deploy-external-cluster
release-deploy-external-cluster: ## Build+push images and deploy using the external-cluster Helm profile
	@echo "$(GREEN)External cluster release deploy with tag $(IMAGE_TAG)$(NC)"
	@test -n "$(IMAGE_REGISTRY)" || { echo "$(RED)IMAGE_REGISTRY is required. Example: make release-deploy-external-cluster IMAGE_REGISTRY=registry.gitlab.com/s4cna/image-factory$(NC)"; exit 1; }
	@$(MAKE) docker-build-all-multiarch IMAGE_REGISTRY=$(IMAGE_REGISTRY) IMAGE_TAG=$(IMAGE_TAG) CONTAINER_ENGINE=$(CONTAINER_ENGINE) PLATFORMS=$(PLATFORMS) FRONTEND_USE_LOCAL_DIST=$(FRONTEND_USE_LOCAL_DIST)
	@if [ "$(OLLAMA_ENABLED)" = "true" ] && [ "$(OLLAMA_STORAGE_MODE)" = "baked" ]; then \
		$(MAKE) docker-build-ollama-baked-push IMAGE_REGISTRY=$(IMAGE_REGISTRY) IMAGE_TAG=$(IMAGE_TAG) CONTAINER_ENGINE=$(CONTAINER_ENGINE) OLLAMA_MODEL_STORE="$(OLLAMA_MODEL_STORE)" OLLAMA_MODEL_NAME="$(OLLAMA_MODEL_NAME)" OLLAMA_BASE_IMAGE="$(OLLAMA_BASE_IMAGE)" OLLAMA_IMAGE_REPOSITORY="$(OLLAMA_IMAGE_REPOSITORY)"; \
	fi
	helm upgrade --install $(HELM_RELEASE) $(HELM_CHART) -n $(HELM_NAMESPACE) \
		-f deploy/helm/image-factory/values.external-cluster.example.yaml \
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
		--set workers.internalRegistryGc.image.tag=$(IMAGE_TAG) \
		--set loki.image.repository=$(IMAGE_REGISTRY)/image-factory-loki \
		--set loki.image.tag=3.0.0 \
		--set alloy.image.repository=$(IMAGE_REGISTRY)/image-factory-alloy \
		--set alloy.image.tag=v1.6.1 \
		--set ollama.enabled=$(OLLAMA_ENABLED) \
		--set ollama.storage.mode=$(OLLAMA_STORAGE_MODE) \
		--set ollama.image.repository=$(OLLAMA_IMAGE_REPOSITORY) \
		--set ollama.image.tag=$(IMAGE_TAG)
	kubectl -n $(HELM_NAMESPACE) rollout status deployment/$(HELM_RELEASE)-backend --timeout=300s
	kubectl -n $(HELM_NAMESPACE) rollout status deployment/$(HELM_RELEASE)-frontend --timeout=300s
	kubectl -n $(HELM_NAMESPACE) rollout status deployment/$(HELM_RELEASE)-docs --timeout=300s

.PHONY: release-deploy-existing-external-cluster
release-deploy-existing-external-cluster: ## Build+push images and upgrade an existing external-cluster release with --reuse-values
	@echo "$(GREEN)Existing external cluster release deploy with tag $(IMAGE_TAG)$(NC)"
	@test -n "$(IMAGE_REGISTRY)" || { echo "$(RED)IMAGE_REGISTRY is required. Example: make release-deploy-existing-external-cluster IMAGE_REGISTRY=registry.gitlab.com/s4cna/image-factory$(NC)"; exit 1; }
	@$(MAKE) docker-build-all-multiarch IMAGE_REGISTRY=$(IMAGE_REGISTRY) IMAGE_TAG=$(IMAGE_TAG) CONTAINER_ENGINE=$(CONTAINER_ENGINE) PLATFORMS=$(PLATFORMS) FRONTEND_USE_LOCAL_DIST=$(FRONTEND_USE_LOCAL_DIST)
	@if [ "$(OLLAMA_ENABLED)" = "true" ] && [ "$(OLLAMA_STORAGE_MODE)" = "baked" ]; then \
		$(MAKE) docker-build-ollama-baked-push IMAGE_REGISTRY=$(IMAGE_REGISTRY) IMAGE_TAG=$(IMAGE_TAG) CONTAINER_ENGINE=$(CONTAINER_ENGINE) OLLAMA_MODEL_STORE="$(OLLAMA_MODEL_STORE)" OLLAMA_MODEL_NAME="$(OLLAMA_MODEL_NAME)" OLLAMA_BASE_IMAGE="$(OLLAMA_BASE_IMAGE)" OLLAMA_IMAGE_REPOSITORY="$(OLLAMA_IMAGE_REPOSITORY)"; \
	fi
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
		--set workers.internalRegistryGc.image.tag=$(IMAGE_TAG) \
		--set loki.enabled=true \
		--set loki.image.repository=$(IMAGE_REGISTRY)/image-factory-loki \
		--set loki.image.tag=3.0.0 \
		--set alloy.enabled=true \
		--set alloy.image.repository=$(IMAGE_REGISTRY)/image-factory-alloy \
		--set alloy.image.tag=v1.6.1 \
		--set ollama.enabled=$(OLLAMA_ENABLED) \
		--set ollama.storage.mode=$(OLLAMA_STORAGE_MODE) \
		--set ollama.image.repository=$(OLLAMA_IMAGE_REPOSITORY) \
		--set ollama.image.tag=$(IMAGE_TAG)
	kubectl -n $(HELM_NAMESPACE) rollout status deployment/$(HELM_RELEASE)-backend --timeout=300s
	kubectl -n $(HELM_NAMESPACE) rollout status deployment/$(HELM_RELEASE)-frontend --timeout=300s
	kubectl -n $(HELM_NAMESPACE) rollout status deployment/$(HELM_RELEASE)-docs --timeout=300s

.PHONY: release-deploy-docs-existing-external-cluster
release-deploy-docs-existing-external-cluster: ## Build+push docs image and upgrade existing external-cluster release with --reuse-values
	@echo "$(GREEN)Existing external cluster docs deploy with tag $(IMAGE_TAG)$(NC)"
	@test -n "$(IMAGE_REGISTRY)" || { echo "$(RED)IMAGE_REGISTRY is required. Example: make release-deploy-docs-existing-external-cluster IMAGE_REGISTRY=registry.gitlab.com/s4cna/image-factory$(NC)"; exit 1; }
	@$(MAKE) docker-build-docs-multiarch IMAGE_REGISTRY=$(IMAGE_REGISTRY) IMAGE_TAG=$(IMAGE_TAG) CONTAINER_ENGINE=$(CONTAINER_ENGINE) PLATFORMS=$(PLATFORMS)
	helm upgrade --install $(HELM_RELEASE) $(HELM_CHART) -n $(HELM_NAMESPACE) --reuse-values \
		--set docs.image.repository=$(IMAGE_REGISTRY)/image-factory-docs \
		--set docs.image.tag=$(IMAGE_TAG) \
		--set docs.image.pullPolicy=Always
	kubectl -n $(HELM_NAMESPACE) rollout status deployment/$(HELM_RELEASE)-docs --timeout=300s

.PHONY: release-deploy-docs-external-cluster
release-deploy-docs-external-cluster: ## Build+push docs image and deploy external-cluster profile (no --reuse-values)
	@echo "$(GREEN)External cluster docs deploy with tag $(IMAGE_TAG)$(NC)"
	@test -n "$(IMAGE_REGISTRY)" || { echo "$(RED)IMAGE_REGISTRY is required. Example: make release-deploy-docs-external-cluster IMAGE_REGISTRY=registry.gitlab.com/s4cna/image-factory$(NC)"; exit 1; }
	@$(MAKE) docker-build-docs-multiarch IMAGE_REGISTRY=$(IMAGE_REGISTRY) IMAGE_TAG=$(IMAGE_TAG) CONTAINER_ENGINE=$(CONTAINER_ENGINE) PLATFORMS=$(PLATFORMS)
	helm upgrade --install $(HELM_RELEASE) $(HELM_CHART) -n $(HELM_NAMESPACE) \
		-f deploy/helm/image-factory/values.external-cluster.example.yaml \
		--set docs.image.repository=$(IMAGE_REGISTRY)/image-factory-docs \
		--set docs.image.tag=$(IMAGE_TAG) \
		--set docs.image.pullPolicy=Always
	kubectl -n $(HELM_NAMESPACE) rollout status deployment/$(HELM_RELEASE)-docs --timeout=300s

.PHONY: release-verify-external-cluster
release-verify-external-cluster: ## Preflight-check external-cluster Helm release inputs before building/pushing
	@test -n "$(IMAGE_REGISTRY)" || { echo "$(RED)IMAGE_REGISTRY is required. Example: make release-verify-external-cluster IMAGE_REGISTRY=registry.gitlab.com/s4cna/image-factory IMAGE_TAG=<tag>$(NC)"; exit 1; }
	@test -n "$(IMAGE_TAG)" || { echo "$(RED)IMAGE_TAG is required. Example: make release-verify-external-cluster IMAGE_REGISTRY=registry.gitlab.com/s4cna/image-factory IMAGE_TAG=v0.1.0-abc123$(NC)"; exit 1; }
	IMAGE_REGISTRY=$(IMAGE_REGISTRY) IMAGE_TAG=$(IMAGE_TAG) HELM_RELEASE=$(HELM_RELEASE) HELM_NAMESPACE=$(HELM_NAMESPACE) HELM_CHART=$(HELM_CHART) ./scripts/verify-external-cluster-release.sh

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
