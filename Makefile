.PHONY: help docker-start docker-stop docker-clean docker-logs docker-shell-postgres docker-shell-mysql \
        build run-postgres run-mysql run-sqlite test test-debug test-trace metrics \
        db-init-sqlite

# Default target
help:
	@echo "Netprobe Exporter - Make Targets"
	@echo "===================================="
	@echo ""
	@echo "Docker Management:"
	@echo "  make docker-start          Start PostgreSQL, MySQL, and test containers"
	@echo "  make docker-stop           Stop all Docker services"
	@echo "  make docker-clean          Remove Docker services and volumes"
	@echo "  make docker-logs           Show Docker logs"
	@echo "  make docker-shell-postgres Open PostgreSQL interactive shell"
	@echo "  make docker-shell-mysql    Open MySQL interactive shell"
	@echo ""
	@echo "Build & Run:"
	@echo "  make build                 Build the exporter binary"
	@echo "  make run-postgres          Build and run with PostgreSQL"
	@echo "  make run-mysql             Build and run with MySQL"
	@echo "  make run-sqlite            Build and run with SQLite"
	@echo ""
	@echo "Database:"
	@echo "  make db-init-sqlite        Initialize local SQLite database"
	@echo ""
	@echo "Testing:"
	@echo "  make test                  Run the test suite (default: minimal output)"
	@echo "  make test-debug            Run tests with debug logging enabled"
	@echo "  make test-trace            Run tests with trace logging (maximum verbosity)"
	@echo "  make metrics               Fetch metrics from running exporter"
	@echo ""

# Docker targets
docker-start:
	@echo "🚀 Starting Docker services..."
	docker-compose up -d
	@echo ""
	@echo "⏳ Waiting for services to be healthy..."
	@sleep 5
	docker-compose ps
	@echo ""
	@echo "✅ Docker services started!"
	@echo ""
	@echo "Next steps:"
	@echo "  make build                 Build the exporter"
	@echo "  make run-postgres          Run with PostgreSQL"

docker-stop:
	@echo "🛑 Stopping Docker services..."
	docker-compose down
	@echo "✅ Docker services stopped"

docker-clean:
	@echo "🧹 Removing Docker services and volumes..."
	docker-compose down -v
	@echo "✅ Docker cleanup complete"

docker-logs:
	docker-compose logs -f

docker-shell-postgres:
	@echo "🔧 Opening PostgreSQL shell..."
	docker exec -it pingding-postgres psql -U exporter_user -d network_monitoring

docker-shell-mysql:
	@echo "🔧 Opening MySQL shell..."
	docker exec -it pingding-mysql mysql -u exporter_user -pexporter_password network_monitoring

# Build target
build:
	@echo "🔨 Building exporter daemon..."
	go build -o netprobe ./cmd/netprobe
	@echo "🔨 Building CLI tool..."
	go build -o netprobe-ping ./cmd/netprobe-ping
	@echo "✅ Build complete: ./netprobe (daemon) and ./netprobe-ping (CLI)"
	@echo ""
	@echo "🔐 Setting CAP_NET_RAW capability..."
	@sudo setcap cap_net_raw=ep ./netprobe 2>/dev/null && echo "   ✅ netprobe" || echo "   ⚠️  netprobe (requires: sudo setcap cap_net_raw=ep ./netprobe)"
	@sudo setcap cap_net_raw=ep ./netprobe-ping 2>/dev/null && echo "   ✅ netprobe-ping" || echo "   ⚠️  netprobe-ping (requires: sudo setcap cap_net_raw=ep ./netprobe-ping)"

# Database initialization
db-init-sqlite:
	@echo "📦 Initializing SQLite database..."
	bash scripts/init-sqlite.sh
	@echo "✅ SQLite database ready"

# Run targets
run-postgres: build
	@echo "🏃 Running exporter with PostgreSQL..."
	./netprobe -config config-postgres.yaml

run-mysql: build
	@echo "🏃 Running exporter with MySQL..."
	./netprobe -config config-mysql.yaml

run-sqlite: db-init-sqlite build
	@echo "🏃 Running exporter with SQLite..."
	./netprobe -config config-sqlite.yaml

# Testing targets
test:
	@echo "🧪 Running tests..."
	go test -v ./tests

test-debug:
	@echo "🐛 Running tests with debug logging..."
	LOG_LEVEL=debug go test -count=1 -v ./tests

test-trace:
	@echo "🔍 Running tests with trace logging (maximum verbosity)..."
	LOG_LEVEL=trace go test -count=1 -v ./tests

metrics:
	@echo "📊 Fetching metrics from exporter..."
	@curl -s http://localhost:9090/metrics | grep netprobe | head -20

# Utility targets for common workflows
setup: docker-start build
	@echo ""
	@echo "🎉 Setup complete! To start the exporter:"
	@echo "  make run-postgres   # Use PostgreSQL"
	@echo "  make run-mysql      # Use MySQL"
	@echo "  make run-sqlite     # Use SQLite"

dev: docker-start
	@echo "🚀 Development environment ready"
	@echo ""
	@echo "In another terminal, run:"
	@echo "  make run-postgres"
	@echo ""
	@echo "Then in a third terminal:"
	@echo "  make metrics"

clean: docker-clean
	@echo "🧹 Cleaning up..."
	rm -f netprobe netprobe-ping netprobe.db
	@echo "✅ Clean complete"
