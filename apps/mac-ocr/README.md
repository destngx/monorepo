# mac-ocr

A high-performance, native macOS CLI utility for document text extraction. It leverages Apple's **Vision** and **PDFKit** frameworks to provide on-device OCR and vector text extraction with zero external dependencies.

## Features

- **Hybrid Extraction**: Automatically detects native PDF text layers and only falls back to OCR when necessary.
- **Hardware Accelerated**: Uses the Apple Neural Engine (ANE) via the Vision framework.
- **Concurrent Processing**: Multi-threaded page processing for large documents.
- **Structured JSON**: Versioned output schema including confidence scores and bounding boxes.
- **Privacy First**: 100% local processing; no data leaves your machine.

## Prerequisites

- macOS 12.0 or later (Monterey+)
- Xcode Command Line Tools (`swiftc` and `make`)

## Installation

To build the binary locally:

```bash
cd apps/mac-ocr
make build
```

The compiled binary will be available at `bin/mac-ocr`.

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

# Specify languages (defaults to en-US)
./bin/mac-ocr document.pdf --lang en-US,vi-VN

# Force OCR even if a text layer exists
./bin/mac-ocr native.pdf --force-ocr

# Process a specific page
./bin/mac-ocr report.pdf --page 5
```

## JSON Schema (v1.0)

The `--format json` flag produces a structured response:

```json
{
  "metadata": {
    "filename": "invoice.pdf",
    "page_count": 1,
    "processed_at": "2024-05-08T..."
  },
  "pages": [
    {
      "index": 1,
      "method": "ocr", // or "direct"
      "content": "extracted text...",
      "confidence": 0.98,
      "bbox": [...] // if --bbox is used
    }
  ],
  "summary": {
    "total_chars": 1250,
    "avg_confidence": 0.98
  }
}
```

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

- `0`: Success
- `1`: Usage/Argument error
- `2`: File not found
- `3`: Unsupported file type
- `4`: OCR engine failure
- `5`: PDF rendering failure
- `6`: Timeout exceeded
