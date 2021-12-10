# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOINSTALL=$(GOCMD) install
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD)get
MODULE_NAME=main
BINARY_NAME=tt
PROJECT_PATH=github.com/matt-steen/todo-tracker
EXE_PATH=$(GOPATH)/bin

# all: test build
all: build

build:
	$(GOBUILD) -o $(EXE_PATH)/$(BINARY_NAME) $(PROJECT_PATH)/cmd/$(MODULE_NAME)

test: 
	$(GOTEST) -v ./...

clean: 
	$(GOCLEAN)
	rm -f $(EXE_PATH)/$(BINARY_NAME)

lint:
	golangci-lint run

format:
	gofmt -w .

run:
	$(BINARY_NAME)
