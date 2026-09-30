NAME = Posturify
XCODE_DEV_DIR ?= /Applications/Xcode.app/Contents/Developer
export DEVELOPER_DIR = $(XCODE_DEV_DIR)

APP_BUNDLE = build/$(NAME).app
RELEASE_BIN = .build/arm64-apple-macosx/release/$(NAME)

.PHONY: all build run test clean release bundle

all: build

build:
	swift build

release:
	swift build -c release

bundle: release
	@mkdir -p $(APP_BUNDLE)/Contents/MacOS
	@mkdir -p $(APP_BUNDLE)/Contents/Resources
	@cp Sources/Resources/Info.plist $(APP_BUNDLE)/Contents/Info.plist
	@cp Resources/AppIcon.icns $(APP_BUNDLE)/Contents/Resources/AppIcon.icns
	@cp $(RELEASE_BIN) $(APP_BUNDLE)/Contents/MacOS/$(NAME)
	@chmod +x $(APP_BUNDLE)/Contents/MacOS/$(NAME)
	@codesign --force --deep --sign - $(APP_BUNDLE)
	@echo "Created $(APP_BUNDLE)"

run:
	swift run

test:
	swift test

clean:
	rm -rf .build build
