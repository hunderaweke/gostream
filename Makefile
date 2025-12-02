# Project Variables
PROJECT_NAME := gostream
PROTO_SRC := internal/proto
GEN_DEST := gen/go/auth
THIRD_PARTY := third_party

# Colors for terminal output
GREEN  := $(shell tput -Txterm setaf 2)
YELLOW := $(shell tput -Txterm setaf 3)
RESET  := $(shell tput -Txterm sgr0)

.PHONY: all setup gen run clean help

# Default target
all: help

## help: Show this help message
help:
	@echo 'Usage:'
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' |  sed -e 's/^/ /'

## setup: Install necessary tools and download google protos
setup:
	@echo "${YELLOW}Installing tools...${RESET}"
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
	go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway@latest
	go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2@latest
	@echo "${YELLOW}Downloading Google API definitions...${RESET}"
	@mkdir -p $(THIRD_PARTY)/google/api
	curl -s -o $(THIRD_PARTY)/google/api/annotations.proto https://raw.githubusercontent.com/googleapis/googleapis/master/google/api/annotations.proto
	curl -s -o $(THIRD_PARTY)/google/api/http.proto https://raw.githubusercontent.com/googleapis/googleapis/master/google/api/http.proto
	@echo "${GREEN}Setup complete!${RESET}"

## gen: Generate Go code from proto files
gen:
	@echo "${YELLOW}Generating gRPC and Gateway code...${RESET}"
	@mkdir -p $(GEN_DEST)
	protoc \
		--proto_path=$(PROTO_SRC) \
		--proto_path=$(THIRD_PARTY) \
		--go_out=$(GEN_DEST) --go_opt=paths=source_relative \
		--go-grpc_out=$(GEN_DEST) --go-grpc_opt=paths=source_relative \
		--grpc-gateway_out=$(GEN_DEST) --grpc-gateway_opt=paths=source_relative \
		$(PROTO_SRC)/auth.proto
	@echo "${GREEN}Generation done! Files in $(GEN_DEST)${RESET}"

## run: Run the application (gRPC + Gateway)
run:
	@echo "${YELLOW}Starting $(PROJECT_NAME)...${RESET}"
	go run cmd/api/main.go

## clean: Remove generated files
clean:
	@echo "${YELLOW}Cleaning generated files...${RESET}"
	rm -rf gen/
	@echo "${GREEN}Cleaned.${RESET}"

## tidy: Tidy go modules
tidy:
	go mod tidy