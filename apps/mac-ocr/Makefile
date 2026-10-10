BINARY_NAME = mac-ocr
BIN_DIR     = bin

build:
	swift build -c release
	@mkdir -p $(BIN_DIR)
	@cp "$$(swift build -c release --show-bin-path)/$(BINARY_NAME)" $(BIN_DIR)/$(BINARY_NAME)
	@echo "✓ Built: $(BIN_DIR)/$(BINARY_NAME)"

test:
	swift test

clean:
	rm -rf $(BIN_DIR) .build

.PHONY: build test clean
