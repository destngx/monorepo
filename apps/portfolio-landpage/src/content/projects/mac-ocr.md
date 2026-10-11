---
title: mac-ocr
summary: A native macOS CLI that extracts text from PDFs and images, using the PDF text layer when it is usable and Apple Vision OCR otherwise. Outputs plain text or JSON for scripts and agents.
year: 2026
status: shipped
stack: [Swift, Vision, PDFKit, ImageIO, SwiftPM]
repo: https://github.com/destngx/monorepo/tree/main/apps/mac-ocr
featured: false
order: 5
sfx: カリッ
---

## The problem

I needed text out of PDFs and scanned documents, in a form a workflow could consume, without sending files to a hosted model. Asking an LLM to read a page through vision costs tokens, is slower, and can drop or invent numbers. I wanted something deterministic that runs locally and returns the text along with its position on the page.

## What it does

- Reads PDFs and common image formats. Each input file produces one result with per-page content.
- Uses the embedded text layer when it looks usable and falls back to OCR for pages that do not have one. `--force-ocr` skips the check.
- Prints plain text or structured JSON (`--format json`). `--bbox` adds bounding boxes in normalized, bottom-left-origin coordinates.
- Supports `--lang`, `--dpi` (72 to 600), a per-page `--timeout`, and `--page` to extract a single page.
- Uses distinct exit codes for usage, missing file, unsupported type, OCR, render, and timeout errors. A failed page carries an `error` field instead of aborting the whole file.

## How it's built

I built it as a SwiftPM package with three targets. `MacOCRCore` holds the argument parser, extractor, PDF pipeline, Vision wrapper, and formatters. The `mac-ocr` executable is a thin entry point that calls into the core, and the XCTest target covers the parser, extractor, and output.

I route files by content type with UniformTypeIdentifiers. PDF pages are rendered to an 8-bit grayscale bitmap at the requested DPI, honoring the crop box and page rotation, and then passed to `VNRecognizeTextRequest` at the `.accurate` level. The minimum platform is macOS 13.

## Interesting bits

- PDFKit is not thread-safe, so every `PDFDocument` and `PDFPage` access goes through one lock. Recognition runs outside that lock.
- Vision serializes recognition internally. The docs record that parallel requests gave no speedup, so I kept the pipeline at two workers: one renders a page while the other is recognized.

```swift
/// Vision serializes text recognition internally, so wider parallelism only multiplies memory.
/// Two workers let one page render while another is recognized.
static let pipelineWidth = 2
```

- The text-layer check requires at least 50 printable characters, with at least 40% of them letters or digits.
- `--timeout` is enforced by a watchdog that cancels the Vision request, which maps to exit code 6.
- The docs record about 0.75 seconds per OCR page at 150 DPI on Apple Silicon, and roughly 150 MB peak for a 12-page scanned PDF.

## What's next

The repo has no roadmap. The docs say the pipeline width should not be raised without benchmarking first, so any change there starts with measurement.
