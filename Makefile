.PHONY: build test bench throughput clean run demo help

# Default compound word components dictionary
COMPONENTS := dictionaries/german_compound_word_components.txt

help:
	@echo "German Tokenizer - Available Commands"
	@echo ""
	@echo "  make build       Build all binaries"
	@echo "  make test        Run unit tests"
	@echo "  make bench       Run Go micro-benchmarks (per-function timing)"
	@echo "  make throughput  Run throughput test (words/sec on test corpus)"
	@echo "  make demo        Run interactive tokenizer demo"
	@echo "  make clean       Remove binaries and generated files"
	@echo ""
	@echo "  make run TEXT=\"your text\"   Tokenize text"
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
