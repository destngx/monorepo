# Native OCR (mac-ocr) — Knowledge Base

This document provides a unified reference for the architecture, usage, and AI workflow integration of the native macOS OCR utility.

## 🏛 Architecture & Engineering

`mac-ocr` is a high-performance Swift utility that interfaces directly with Apple's low-level frameworks:

- **Vision Framework**: Uses the Apple Neural Engine (ANE) via `VNRecognizeTextRequest` (latest revision, `.accurate`).
- **PDFKit**: Handles document traversal and grayscale crop-box rendering (default 150 DPI, page rotation honored).
- **Hybrid Logic**: Automatically extracts native vector text (`direct` path) and fallbacks to OCR (`ocr` path) only when the text layer is insufficient or missing.

### Code Layout (SwiftPM)

- `Sources/MacOCRCore/`: `CommandLineParser`, `Extractor` (routing + images), `PDFPipeline`, `TextRecognizer` (Vision + timeout), `OutputFormatter`, `Models`, `OCRError` (exit codes).
- `Sources/mac-ocr/main.swift`: thin entry point. Keep logic in `MacOCRCore` so it stays testable.
- `Tests/MacOCRCoreTests/`: XCTest; fixtures are generated at runtime by `Fixtures.swift`.

### Concurrency Rules

- **PDFKit is not thread-safe**: all `PDFDocument`/`PDFPage` access goes through the pipeline's document lock.
- **Vision serializes text recognition internally**: parallel requests give no speedup (measured), so PDF pages use a 2-wide pipeline (render one page while the other is recognized) and files run sequentially. Do not widen this without benchmarking; it only multiplies memory.
- **`PDFPage.draw(with:to:)` already applies `transform(for:)`** (box origin + rotation). Never concatenate it again.

### Performance Profile

- **Latency**: ~0.75s per OCR page at 150 DPI on Apple Silicon (Vision-bound); text-layer PDFs are near-instant.
- **Memory**: ~150 MB peak for a 12-page scanned PDF (bounded by the pipeline width, not page count).
- **Privacy**: 100% on-device. Zero data leaves the local machine.

## 🚀 Usage Guide

### Building the Tool

The binary must be compiled locally for your macOS architecture:

```bash
cd apps/mac-ocr
make build
```

_Binary output: `apps/mac-ocr/bin/mac-ocr`_ (also `pnpx nx build mac-ocr`; tests: `pnpx nx test mac-ocr`)

### CLI Commands

```bash
# Standard text output
./apps/mac-ocr/bin/mac-ocr sample.pdf

# Structured JSON (Recommended for Agents)
./apps/mac-ocr/bin/mac-ocr sample.pdf --format json --bbox
```

| Flag          | Description                                              | Default |
| :------------ | :------------------------------------------------------- | :------ |
| `--format`    | Output mode (`plain` or `json`)                          | `plain` |
| `--lang`      | Comma-separated Vision codes (e.g. `vi-VT`, not `vi-VN`) | `en-US` |
| `--dpi`       | Render resolution (72-600)                               | `150`   |
| `--force-ocr` | Skip direct extraction check                             | `false` |
| `--bbox`      | Include spatial metadata                                 | `false` |
| `--timeout`   | Per-page recognition timeout (seconds)                   | `30`    |
| `--page`      | Extract only page N (PDFs)                               | all     |

## 🔗 AI Workflow Integration (GraphWeave)

The tool is designed to be a reliable **CLI Node** primitive. This allows you to extract intelligence deterministically before passing it to an LLM.

### Why use this over LLM Vision?

- **Cost**: $0/token.
- **Spatial Awareness**: `--bbox` provides normalized coordinates (bottom-left origin) for table and header detection.
- **Deterministic**: No hallucinations in text extraction; perfectly preserves numbers and technical terms.

### Example Node Definition

```json
{
  "id": "document_ingest",
  "type": "cli_node",
  "config": {
    "command": "apps/mac-ocr/bin/mac-ocr {{file_path}} --format json --bbox",
    "output_key": "ocr_result"
  }
}
```

### Accessing Data in Downstream Nodes

```
{{document_ingest_output.ocr_result.pages[0].content}}
{{document_ingest_output.ocr_result.summary.total_chars}}
```

## 🛠 Maintenance & Troubleshooting

- **Requirement**: macOS 13.0+ (Ventura).
- **Diagnostics**: Errors and per-page warnings are written to `stderr`; successful data to `stdout`. Failed pages carry an `error` field and appear in `summary.warnings`.
- **Exit Codes**: `0` (Success), `1` (Usage), `2` (File Missing), `3` (Unsupported Type), `4` (OCR Error), `5` (Render Error), `6` (Timeout). Non-zero only when a whole file fails.
