.PHONY: help install dev build test test-ui test-run lint clean deploy

# Configuration
DEPLOY_HOST ?= bbb1.wegman

# Default target
help:
	@echo "Available commands:"
	@echo "  make install    - Install dependencies"
	@echo "  make dev        - Start development server"
	@echo "  make build      - Build the application for production"
	@echo "  make test       - Run tests in watch mode"
	@echo "  make test-ui    - Run tests with UI"
	@echo "  make test-run   - Run tests once"
	@echo "  make lint       - Run ESLint"
	@echo "  make clean      - Clean build artifacts"
	@echo "  make deploy     - Build and deploy to \$${DEPLOY_HOST} (default: bbb1.wegman)"

# Install dependencies
install:
	npm install

# Start development server
dev:
	npm run dev

# Build the application
build:
	npm run build

# Run tests in watch mode
test:
	npm test

# Run tests with UI
test-ui:
	npm run test:ui

# Run tests once
test-run:
	npm run test:run

# Run linter
lint:
	npm run lint

# Clean build artifacts
clean:
	rm -rf dist node_modules/.tmp

# Build and deploy to server
deploy:
	@echo "Building for production with API URL: http://$(DEPLOY_HOST):8080..."
	VITE_API_BASE_URL=http://$(DEPLOY_HOST):8080 npm run build
	@echo "Deploying to $(DEPLOY_HOST):/usr/share/nginx/html..."
	rsync -avz --delete dist/ $(DEPLOY_HOST):/usr/share/nginx/html/
	@echo "Deployment complete!"
