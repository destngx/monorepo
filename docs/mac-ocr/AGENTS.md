# Native OCR (mac-ocr) — Knowledge Base

This document provides a unified reference for the architecture, usage, and AI workflow integration of the native macOS OCR utility.

## 🏛 Architecture & Engineering

`mac-ocr` is a high-performance Swift utility that interfaces directly with Apple's low-level frameworks:

- **Vision Framework**: Uses the Apple Neural Engine (ANE) via `VNRecognizeTextRequest` (Revision 3).
- **PDFKit**: Handles document traversal and 150 DPI rendering.
- **Hybrid Logic**: Automatically extracts native vector text (`direct` path) and fallbacks to OCR (`ocr` path) only when the text layer is insufficient or missing.

### Performance Profile

- **Latency**: ~150ms for images; ~200ms for searchable PDFs; ~250ms/page for scanned PDFs.
- **Privacy**: 100% on-device. Zero data leaves the local machine.

## 🚀 Usage Guide

### Building the Tool

The binary must be compiled locally for your macOS architecture:

```bash
cd apps/mac-ocr
make build
```

_Binary output: `apps/mac-ocr/bin/mac-ocr`_

### CLI Commands

```bash
# Standard text output
./apps/mac-ocr/bin/mac-ocr sample.pdf

# Structured JSON (Recommended for Agents)
./apps/mac-ocr/bin/mac-ocr sample.pdf --format json --bbox
```

| Flag          | Description                     | Default |
| :------------ | :------------------------------ | :------ |
| `--format`    | Output mode (`plain` or `json`) | `plain` |
| `--lang`      | Comma-separated ISO codes       | `en-US` |
| `--dpi`       | Render resolution (72-600)      | `150`   |
| `--force-ocr` | Skip direct extraction check    | `false` |
| `--bbox`      | Include spatial metadata        | `false` |

## 🔗 AI Workflow Integration (GraphWeave)

The tool is designed to be a reliable **CLI Node** primitive. This allows you to extract intelligence deterministically before passing it to an LLM.

### Why use this over LLM Vision?

- **Cost**: $0/token.
- **Spatial Awareness**: `--bbox` provides normalized coordinates for table and header detection.
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

- **Requirement**: macOS 12.0+ (Monterey) for best accuracy.
- **Diagnostics**: Errors are written to `stderr`; successful data to `stdout`.
- **Exit Codes**: `0` (Success), `2` (File Missing), `4` (OCR Error), `5` (Render Error).
