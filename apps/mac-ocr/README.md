# mac-ocr

A high-performance, native macOS CLI utility for document text extraction. It leverages Apple's **Vision** and **PDFKit** frameworks to provide on-device OCR and vector text extraction with zero external dependencies.

## Features

- **Hybrid Extraction**: Automatically detects native PDF text layers and only falls back to OCR when necessary.
- **Hardware Accelerated**: Uses the Apple Neural Engine (ANE) via the Vision framework.
- **Pipelined Processing**: Renders the next PDF page while the current one is recognized, with bounded memory.
- **Orientation Aware**: Honors PDF page rotation, crop boxes, and image EXIF orientation.
- **Structured JSON**: Versioned output schema including confidence scores, bounding boxes, and per-page errors.
- **Privacy First**: 100% local processing; no data leaves your machine.

## Prerequisites

- macOS 13.0 or later (Ventura+)
- Xcode (Swift 5.9+; XCTest is required for `make test`)

## Installation

To build the binary locally:

```bash
cd apps/mac-ocr
make build   # or: pnpx nx build mac-ocr
make test    # or: pnpx nx test mac-ocr
```

The compiled binary will be available at `bin/mac-ocr`.

## Layout

- `Sources/MacOCRCore`: argument parsing, PDF/image pipelines, Vision bridge, output formatting
- `Sources/mac-ocr`: thin executable entry point
- `Tests/MacOCRCoreTests`: XCTest suite; fixtures are generated at runtime

## Usage

### Basic Extraction

```bash
./bin/mac-ocr document.pdf
```

### JSON Output with Bounding Boxes

```bash
./bin/mac-ocr invoice.jpg --format json --bbox
```

### Advanced Options

```bash
# Set custom DPI for higher accuracy on small text
./bin/mac-ocr scan.pdf --dpi 300

# Specify languages (defaults to en-US; unsupported codes are rejected with the supported list)
./bin/mac-ocr document.pdf --lang en-US,vi-VT

# Fail a page if recognition takes longer than 10 seconds
./bin/mac-ocr scan.pdf --timeout 10

# Force OCR even if a text layer exists
./bin/mac-ocr native.pdf --force-ocr

# Process a specific page
./bin/mac-ocr report.pdf --page 5
```

## JSON Schema (v1.1)

The `--format json` flag produces one object per file (an array when several files are given):

```json
{
  "schema_version": "1.1",
  "metadata": {
    "filename": "invoice.pdf",
    "file_size_bytes": 48211,
    "page_count": 1,
    "processed_at": "2026-10-10T...",
    "tool_version": "1.1.0",
    "config": { "dpi": 150, "languages": ["en-US"], "force_ocr": false }
  },
  "pages": [
    {
      "index": 1,
      "method": "ocr", // or "direct"
      "content": "extracted text...",
      "confidence": 0.98,
      "char_count": 1250,
      "bbox": [{ "text": "...", "confidence": 0.99, "x": 0.1, "y": 0.8, "w": 0.5, "h": 0.02 }], // with --bbox
      "error": "..." // only when the page failed; content is then empty
    }
  ],
  "summary": {
    "total_chars": 1250,
    "pages_direct": 0,
    "pages_ocr": 1,
    "avg_confidence": 0.98,
    "warnings": [] // "page N: <error>" for each failed page
  }
}
```

Bounding boxes are normalized (0-1) with a **bottom-left origin** (Vision convention): larger `y` is higher on the page.

v1.1 changes: failed pages report `error` with empty `content` (previously an `[Error: ...]` string in `content`), `summary.warnings` is populated, and bbox entries carry `confidence`.

## Integration with GraphWeave

Use `mac-ocr` as a `cli_node` in your workflows for deterministic, cost-effective document ingestion.

```json
{
  "id": "ingest_pdf",
  "type": "cli_node",
  "config": {
    "command": "apps/mac-ocr/bin/mac-ocr {{input.file_path}} --format json",
    "output_key": "ocr_data"
  }
}
```

## Exit Codes

If some pages of a document fail, the command still succeeds and reports them in `warnings` (and on stderr). If every page fails, it exits with that failure's code.

- `0`: Success
- `1`: Usage/Argument error
- `2`: File not found
- `3`: Unsupported file type
- `4`: OCR engine failure
- `5`: PDF rendering failure
- `6`: Timeout exceeded
