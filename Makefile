.PHONY: build test bench throughput clean run demo help dict-stats dict-contains dict-add dict-remove

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
	@echo "  make run TEXT=\"your text\"           Tokenize text"
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
