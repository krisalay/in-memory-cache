# ================================
# Project Configuration
# ================================

APP_NAME := in-memory-cache
PKG := ./...
GO := go
GOFLAGS := -race
BENCH_FLAGS := -bench=. -benchmem
PROFILE_DIR := profiles

# ================================
# Default Target
# ================================

.PHONY: help
help:
	@echo ""
	@echo "Available commands:"
	@echo "----------------------------------------"
	@echo "make run           → Run demo (main.go)"
	@echo "make test          → Run all tests"
	@echo "make bench         → Run benchmarks"
	@echo "make bench-save    → Run benchmarks and save output"
	@echo "make load          → Run concurrency load test (benchmark.go)"
	@echo "make fmt           → Format code"
	@echo "make lint          → Run static analysis"
	@echo "make clean         → Clean build artifacts"
	@echo "----------------------------------------"
	@echo ""

# ================================
# Run Demo
# ================================

.PHONY: run
run:
	@echo "Running cache demo..."
	$(GO) run cmd/main.go

# ================================
# Run Load Benchmark (Real Concurrency)
# ================================

.PHONY: load
load:
	@echo "Running concurrency load test..."
	$(GO) run cmd/benchmark/benchmark.go

# ================================
# Tests
# ================================

.PHONY: test
test:
	@echo "Running tests..."
	$(GO) test  -v

# ================================
# Benchmarks
# ================================

.PHONY: bench
bench:
	@echo "Running benchmarks..."
	$(GO) test $(PKG) $(BENCH_FLAGS)

.PHONY: bench-save
bench-save:
	@echo "Saving benchmark results..."
	mkdir -p $(PROFILE_DIR)
	$(GO) test $(PKG) $(BENCH_FLAGS) | tee $(PROFILE_DIR)/bench.txt

.PHONY: clean
clean:
	@echo "Cleaning artifacts..."
	rm -rf $(PROFILE_DIR)
