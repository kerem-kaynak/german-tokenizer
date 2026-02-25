.PHONY: build test benchmark clean run demo help

# Default compound word components dictionary
COMPONENTS := dictionaries/german_compound_word_components.txt

help:
	@echo "German Tokenizer - Available Commands"
	@echo ""
	@echo "  make build      Build all binaries"
	@echo "  make test       Run unit tests"
	@echo "  make benchmark  Run performance benchmarks"
	@echo "  make demo       Run interactive tokenizer demo"
	@echo "  make clean      Remove binaries and generated files"
	@echo ""
	@echo "  make run TEXT=\"your text\"   Tokenize text"
	@echo ""

build:
	@echo "Building binaries..."
	@mkdir -p bin
	@go build -o bin/tokenize ./cmd/tokenize
	@go build -o bin/dictmgr ./cmd/dictmgr
	@go build -o bin/benchmark ./cmd/benchmark
	@echo "Done. Binaries in ./bin/"

test:
	@echo "Running tests..."
	@go test ./pkg/tokenizer/... -v

benchmark: build
	@./bin/benchmark $(COMPONENTS)

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
