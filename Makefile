.PHONY: all build test bench throughput clean run demo help dict-stats dict-contains dict-add dict-remove deps fmt lint check cover install ci

# Default compound word components dictionary
COMPONENTS := dictionaries/german_compound_word_components.txt

all: build test

help:
	@echo "German Tokenizer - Available Commands"
	@echo ""
	@echo "Build & Run:"
	@echo "  make build       Build all binaries to ./bin/"
	@echo "  make install     Install binaries to GOPATH/bin"
	@echo "  make clean       Remove binaries and generated files"
	@echo "  make run TEXT=\"your text\"   Tokenize text"
	@echo "  make demo        Run interactive tokenizer demo"
	@echo ""
	@echo "Testing & Quality:"
	@echo "  make test        Run unit tests"
	@echo "  make bench       Run Go micro-benchmarks"
	@echo "  make throughput  Run throughput test (words/sec)"
	@echo "  make cover       Run tests with coverage report"
	@echo "  make check       Run all checks (fmt, lint, test)"
	@echo "  make ci          Run CI pipeline locally"
	@echo ""
	@echo "Code Quality:"
	@echo "  make fmt         Format code"
	@echo "  make lint        Run go vet"
	@echo "  make check-fmt   Check if code is formatted"
	@echo "  make deps        Install/update dependencies"
	@echo ""
	@echo "Dictionary Management:"
	@echo "  make dict-stats                     Show dictionary statistics"
	@echo "  make dict-contains WORD=haus        Check if word exists"
	@echo "  make dict-add WORD=neueswort        Add word to dictionary"
	@echo "  make dict-remove WORD=alteswort     Remove word from dictionary"
	@echo ""

build:
	@echo "Building binaries..."
	@mkdir -p bin
	@go build -o bin/tokenize ./cmd/tokenize
	@go build -o bin/dictmgr ./cmd/dictmgr
	@go build -o bin/throughput ./cmd/throughput
	@echo "Done. Binaries in ./bin/"

test:
	@echo "Running tests..."
	@go test ./pkg/tokenizer/... -v

bench:
	@echo "Running Go micro-benchmarks..."
	@go test -bench=. -benchmem ./pkg/tokenizer/...

throughput: build
	@./bin/throughput $(COMPONENTS)

demo: build
	@./bin/tokenize $(COMPONENTS)

run: build
	@./bin/tokenize $(COMPONENTS) "$(TEXT)"

# Dictionary management
dict-stats: build
	@./bin/dictmgr $(COMPONENTS) stats

dict-contains: build
	@./bin/dictmgr $(COMPONENTS) contains $(WORD)

dict-add: build
	@./bin/dictmgr $(COMPONENTS) add $(WORD)

dict-remove: build
	@./bin/dictmgr $(COMPONENTS) remove $(WORD)

clean:
	@rm -rf bin/
	@rm -f dictionaries/*.fst
	@echo "Cleaned."

# Install dependencies
deps:
	@go mod tidy
	@go mod download

# Format code
fmt:
	@go fmt ./...

# Lint code
lint:
	@go vet ./...

# Check formatting (fails if not formatted)
check-fmt:
	@test -z "$$(gofmt -l .)" || (echo "Code is not formatted. Run 'make fmt'" && gofmt -d . && exit 1)

# Run all checks (same as CI)
check: check-fmt lint test

# Run tests with coverage
cover:
	@go test -coverprofile=coverage.out ./pkg/tokenizer/...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

# Install binaries to GOPATH/bin
install:
	@go install ./cmd/tokenize
	@go install ./cmd/dictmgr
	@go install ./cmd/throughput
	@echo "Installed to $(shell go env GOPATH)/bin"

# CI pipeline (what runs in GitHub Actions)
ci: check-fmt lint test
