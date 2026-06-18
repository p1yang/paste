.PHONY: all backend frontend build-backend build-frontend dev test clean pack dist

BACKEND_DIR := backend
FRONTEND_DIR := frontend
BIN_DIR := $(BACKEND_DIR)/bin

all: build

build: build-backend build-frontend

build-backend:
	@echo "Building backend..."
	@mkdir -p $(BIN_DIR)
	cd $(BACKEND_DIR) && CGO_ENABLED=1 GOOS=darwin GOARCH=amd64 go build -o bin/paste-backend cmd/main.go
	@if [ -f "$(BACKEND_DIR)/bin/paste-backend" ]; then \
		echo "Backend built successfully (amd64)"; \
	else \
		cd $(BACKEND_DIR) && CGO_ENABLED=1 go build -o bin/paste-backend cmd/main.go; \
		echo "Backend built successfully (default arch)"; \
	fi

build-frontend:
	@echo "Building frontend..."
	cd $(FRONTEND_DIR) && npm install && npm run build
	@echo "Frontend built successfully"

dev-backend:
	cd $(BACKEND_DIR) && go run cmd/main.go -data-dir ./data

dev-frontend:
	cd $(FRONTEND_DIR) && npm run dev

test-backend:
	@echo "Running backend tests..."
	cd $(BACKEND_DIR) && go test -v ./...

test-frontend:
	@echo "Running frontend tests..."
	cd $(FRONTEND_DIR) && npm test

test: test-backend test-frontend

lint-frontend:
	cd $(FRONTEND_DIR) && npx tsc --noEmit

pack: build
	@echo "Packaging application..."
	cd $(FRONTEND_DIR) && npm run pack

dist: build
	@echo "Building distribution..."
	cd $(FRONTEND_DIR) && npm run dist

clean:
	@echo "Cleaning..."
	rm -rf $(BACKEND_DIR)/bin
	rm -rf $(FRONTEND_DIR)/build
	rm -rf $(FRONTEND_DIR)/release
	rm -rf $(FRONTEND_DIR)/node_modules

install-deps:
	@echo "Installing dependencies..."
	cd $(BACKEND_DIR) && go mod download
	cd $(FRONTEND_DIR) && npm install
