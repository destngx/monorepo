# mac-ocr

A high-performance, native macOS CLI utility for document text extraction. It leverages Apple's **Vision** and **PDFKit** frameworks to provide on-device OCR and vector text extraction with zero external dependencies.

## Features

- **Hybrid Extraction**: Automatically detects native PDF text layers and only falls back to OCR when necessary.
- **Hardware Accelerated**: Uses the Apple Neural Engine (ANE) via the Vision framework.
- **Pipelined Processing**: Renders the next PDF page while the current one is recognized, with bounded memory.
- **Orientation Aware**: Honors PDF page rotation, crop boxes, and image EXIF orientation.
- **Structured JSON**: Versioned output schema including confidence scores, bounding boxes, and per-page errors.
- **Row layout**: Tables come out one row per line with tab-separated cells, even on photos tilted up to 15°. Wrapped prose in side-by-side columns is rejoined per column, and empty table cells get a second, enlarged read (Vision's page pass skips lone tokens like a `%` unit).
- **Masking geometry**: With `--bbox`, every line (text-layer PDF pages included) has its four corners, and barcode/QR regions are reported without their payload.
- **Page images**: `--render-dir` writes each page as an upright sRGB JPEG that the coordinates map onto.
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

# Vision's own line order instead of the row layout
./bin/mac-ocr scan.jpg --layout raw

# Write upright page images to pages/page-<n>.jpg (one input per call)
./bin/mac-ocr report.pdf --format json --bbox --render-dir pages --jpeg-quality 0.85
```

### Text layout

`--layout rows` (the default) builds `content` from line geometry:

1. It estimates the page skew from the line boxes and works in deskewed coordinates.
2. Lines whose vertical bands overlap form one row; cells are sorted left to right and joined with a tab.
3. A line that starts lowercase, sits tight under a long line with the same left edge, and follows a line that does not end a sentence is joined to it, so wrapped paragraphs in two columns stay whole.
4. In a run of rows with three or more cells, rows with the most cells define the columns. A shorter row keeps its cells in their columns and leaves the missing ones empty, after a second read of that cell region at 3x.

`--layout raw` keeps Vision's order, which reads ruled tables column by column.

## JSON Schema (v1.3)

The `--format json` flag produces one object per file (an array when several files are given):

```json
{
  "schema_version": "1.3",
  "metadata": {
    "filename": "invoice.pdf",
    "file_size_bytes": 48211,
    "page_count": 1,
    "processed_at": "2026-10-10T...",
    "tool_version": "1.3.0",
    "config": { "dpi": 150, "languages": ["en-US"], "force_ocr": false, "layout": "rows" }
  },
  "pages": [
    {
      "index": 1,
      "method": "ocr", // or "direct"
      "content": "extracted text...",
      "confidence": 0.98,
      "char_count": 1250,
      "bbox": [{ "text": "...", "confidence": 0.99, "x": 0.1, "y": 0.8, "w": 0.5, "h": 0.02,
                 "quad": [[0.1, 0.82], [0.6, 0.83], [0.6, 0.81], [0.1, 0.8]],
                 "words": [{ "text": "...", "quad": [[x, y], [x, y], [x, y], [x, y]] }] }], // with --bbox
      "barcodes": [{ "symbology": "QR", "quad": [[x, y], [x, y], [x, y], [x, y]] }], // with --bbox
      "skew_degrees": 2.5, // with --bbox, OCR pages
      "image": { "path": "pages/page-1.jpg", "width": 1654, "height": 2339 }, // with --render-dir
      "warnings": ["barcode detection failed: ..."], // only when an optional step failed
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

v1.3 changes (additive): `bbox` entries carry `words`, the line's whitespace-separated tokens in order, each with its own `quad`. For OCR lines the corners come from Vision's box for that token; for text-layer PDF pages they are the union of the token's glyph boxes. Use them to mask part of a line. `words` is absent when a token could not be boxed (for example a ligature that PDFKit reports as several characters); fall back to the line `quad`.

v1.2 changes (all additive):

- `bbox` entries carry `quad`, the four corners in the order top-left, top-right, bottom-right, bottom-left. For OCR lines these are Vision's corners, which follow the text on a tilted photo for longer lines (short cells may come back axis-aligned). For text-layer PDF pages they are the corners of the PDFKit line box. Text-layer pages now get `bbox` with `--bbox` too, at confidence 1.
- `barcodes` lists barcode and QR regions. It is present, possibly empty, whenever detection ran (with `--bbox`). The payload is never output, since it can hold patient IDs.
- `skew_degrees` is the estimated text skew of an OCR page (positive: text rises left to right).
- `image` describes the page JPEG written with `--render-dir`: 8-bit sRGB, EXIF orientation and PDF rotation and crop box applied. Text-layer pages are rendered too. `--render-dir` takes exactly one input; more is a usage error.
- `content` of OCR pages follows `--layout` (default `rows`); `config.layout` records it.
- `warnings` on a page lists optional steps that did not finish, such as barcode detection or the empty-cell re-read. The page text is still valid, and `summary.warnings` repeats them as `page N: ...`. When barcode detection fails, the page has no `barcodes` key.
- A failed page keeps `image` when its JPEG was already written.

v1.1 changes: failed pages report `error` with empty `content` (previously an `[Error: ...]` string in `content`), `summary.warnings` is populated, and bbox entries carry `confidence`.

## Use before an LLM

Run `mac-ocr` in a script to turn images and PDFs into text before an LLM step, so extraction costs nothing and keeps numbers exact:

```sh
apps/mac-ocr/bin/mac-ocr "$file_path" --format json > ocr.json
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
