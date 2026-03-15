BINARY_NAME=imgdup
BIN_DIR=bin

.PHONY: build clean

build:
	@mkdir -p $(BIN_DIR)
	go build -o $(BIN_DIR)/$(BINARY_NAME) .

clean:
	rm -rf $(BIN_DIR)
