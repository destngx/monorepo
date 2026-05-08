BINARY_NAME = mac-ocr
SRC_DIR     = Sources
BIN_DIR     = bin
SOURCES     = $(SRC_DIR)/main.swift \
              $(SRC_DIR)/InputRouter.swift \
              $(SRC_DIR)/PDFPipeline.swift \
              $(SRC_DIR)/VisionBridge.swift \
              $(SRC_DIR)/Formatter.swift \
              $(SRC_DIR)/Models.swift

build:
	@mkdir -p $(BIN_DIR)
	swiftc $(SOURCES) \
	  -framework Vision \
	  -framework PDFKit \
	  -framework AppKit \
	  -framework UniformTypeIdentifiers \
	  -O \
	  -o $(BIN_DIR)/$(BINARY_NAME)
	@echo "✓ Built: $(BIN_DIR)/$(BINARY_NAME)"

clean:
	rm -rf $(BIN_DIR)

.PHONY: build clean
