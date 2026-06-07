.PHONY: help run stop test test-v test-cover clean seed reset build deploy install deps install-admin build-admin dev-admin clean-admin adduser build-css

# Default target: show help
help:
	@echo "Available commands:"
	@echo "  make run          - Start server (for development)"
	@echo "  make stop         - Stop running server"
	@echo "  make test         - Run tests"
	@echo "  make test-v       - Run tests with verbose output"
	@echo "  make test-cover   - Show test coverage"
	@echo "  make clean        - Delete database and admin SPA build artifacts"
	@echo "  make seed         - Insert test data"
	@echo "  make reset        - Reset database and insert test data (also rebuilds admin SPA)"
	@echo "  make build        - Build everything for production (install deps, build frontend, build backend)"
	@echo "  make deploy       - Build and deploy to /opt/goblog (requires sudo)"
	@echo "  make install      - Install npm dependencies (Tailwind CLI)"
	@echo "  make install-admin - Install admin SPA npm dependencies"
	@echo "  make deps         - Download Go dependencies"
	@echo "  make adduser      - Add admin user"
	@echo "  make regenerate-variants - Backfill WebP variants for existing /uploads images"
	@echo "  make build-admin   - Build admin SPA"
	@echo "  make build-css     - Build Tailwind CSS for public pages"
	@echo "  make dev-admin     - Start admin SPA development server"
	@echo "  make clean-admin   - Delete admin SPA build artifacts"

# Start server (builds CSS first)
run: build-css
	@echo "Starting server..."
	go run cmd/goblog/main.go

# Stop server
stop:
	@echo "Stopping goblog process..."
	@lsof -ti :8080 | xargs kill -9 2>/dev/null || true
	@pkill -f "go run cmd/goblog/main.go" 2>/dev/null || true
	@pkill -f "goblog" 2>/dev/null || true
	@echo "Stopped"

# Run tests
test:
	@echo "Running tests..."
	go test ./...

# Run tests with verbose output
test-v:
	@echo "Running tests with verbose output..."
	go test ./... -v

# Show test coverage
test-cover:
	@echo "Calculating test coverage..."
	go test ./... -cover

# Delete database and admin SPA build artifacts
clean: clean-admin
	@echo "Deleting database..."
	@rm -f data/goblog.db
	@echo "Database deleted"

# Insert test data
seed:
	@echo "Inserting test data..."
	go run cmd/seed/main.go

# Add admin user (for production)
adduser:
	@echo "Adding admin user..."
	go run cmd/adduser/main.go

# Backfill WebP variants for existing /uploads images (one-shot after deploy)
regenerate-variants:
	@echo "Regenerating WebP variants under $${UPLOAD_DIR:-data/uploads}..."
	go run cmd/regenerate-variants/main.go

# Reset database and insert test data (also rebuilds admin SPA)
reset: clean build-admin build-css seed

# Build everything for production
build: install install-admin build-admin build-css
	@echo "Building backend..."
	@mkdir -p bin
	go build -o bin/goblog cmd/goblog/main.go
	go build -o bin/seed cmd/seed/main.go
	go build -o bin/adduser cmd/adduser/main.go
	go build -o bin/regenerate-variants cmd/regenerate-variants/main.go
	@echo "Build complete: bin/goblog, bin/seed, bin/adduser, bin/regenerate-variants"

# Build and deploy to /opt/goblog (Linux production server only)
deploy: build
	@echo "Deploying to /opt/goblog..."
	sudo mkdir -p /opt/goblog/bin
	sudo mv bin/goblog bin/adduser bin/seed bin/regenerate-variants /opt/goblog/bin/
	sudo chown root:root /opt/goblog/bin/goblog /opt/goblog/bin/adduser /opt/goblog/bin/seed /opt/goblog/bin/regenerate-variants
	sudo install -m 755 -o root -g root scripts/backup-db.sh /opt/goblog/bin/backup-db.sh
	sudo systemctl daemon-reload
	sudo systemctl restart goblog
	@echo "Deploy complete"

# Install npm dependencies (Tailwind CLI for public pages)
install:
	@echo "Installing npm dependencies (Tailwind CLI)..."
	npm install
	@echo "Installation complete"

# Download dependencies
deps:
	@echo "Downloading dependencies..."
	go mod download
	@echo "Download complete"

# Install admin SPA npm dependencies
install-admin:
	@echo "Installing admin SPA dependencies..."
	cd web-admin && npm install
	@echo "Installation complete"

# Build admin SPA
build-admin:
	@echo "Building admin SPA..."
	cd web-admin && npm run build
	@echo "Admin SPA build complete: web-admin/dist/"

# Start admin SPA development server
dev-admin:
	@echo "Starting admin SPA development server..."
	cd web-admin && npm run dev

# Delete admin SPA build artifacts
clean-admin:
	@echo "Deleting admin SPA build artifacts..."
	@rm -rf web-admin/dist
	@echo "Deletion complete"

# Build Tailwind CSS for public pages
build-css:
	@echo "Building Tailwind CSS for public pages..."
	npm run build:css
	@echo "CSS build complete: internal/view/static/tailwind.css"
