import Foundation
import PDFKit
import Vision

enum PDFPipeline {
    /// Upper bound on the rendered bitmap's longest side, guarding against oversized page boxes at high DPI.
    static let maxRenderDimension: CGFloat = 10_000
    /// Vision serializes text recognition internally, so wider parallelism only multiplies memory.
    /// Two workers let one page render while another is recognized.
    static let pipelineWidth = 2

    static func process(url: URL, options: OCROptions) throws -> [PageResult] {
        guard let document = PDFDocument(url: url) else {
            throw OCRError.renderFailed("Failed to load PDF document: \(url.lastPathComponent)")
        }
        if document.isLocked {
            throw OCRError.renderFailed("PDF is password-protected: \(url.lastPathComponent)")
        }

        let pageIndices = try selectPages(total: document.pageCount, target: options.targetPage)

        // PDFKit is not thread-safe: every PDFDocument/PDFPage access is serialized through `documentLock`.
        let documentLock = NSLock()
        var outcomes = [(page: PageResult, error: OCRError?)?](repeating: nil, count: pageIndices.count)
        let outcomesLock = NSLock()

        let queue = OperationQueue()
        queue.maxConcurrentOperationCount = pipelineWidth
        for (i, pageIndex) in pageIndices.enumerated() {
            queue.addOperation {
                autoreleasepool {
                    let outcome = processPage(document: document, pageIndex: pageIndex, documentLock: documentLock, options: options)
                    outcomesLock.lock()
                    outcomes[i] = outcome
                    outcomesLock.unlock()
                }
            }
        }
        queue.waitUntilAllOperationsAreFinished()

        let completed = outcomes.compactMap { $0 }
        let errors = completed.compactMap(\.error)
        if !completed.isEmpty, errors.count == completed.count {
            throw errors[0].prefixed("All \(completed.count) page(s) failed; page \(completed[0].page.index)")
        }
        return completed.map(\.page)
    }

    static func selectPages(total: Int, target: Int?) throws -> [Int] {
        guard let target else { return Array(0..<total) }
        guard total > 0, target >= 1, target <= total else {
            throw OCRError.usage("Page \(target) is out of range (1-\(total))")
        }
        return [target - 1]
    }

    private static func processPage(
        document: PDFDocument,
        pageIndex: Int,
        documentLock: NSLock,
        options: OCROptions
    ) -> (page: PageResult, error: OCRError?) {
        let number = pageIndex + 1
        let writesImage = options.renderDirectory != nil

        let prepared: Result<PreparedPage, OCRError> = {
            documentLock.lock()
            defer { documentLock.unlock() }
            guard let page = document.page(at: pageIndex) else {
                return .failure(.renderFailed("Failed to load page"))
            }
            var text: String?
            var lines: [BBox]?
            if !options.forceOCR, let layer = page.string, TextLayer.isUsable(layer) {
                text = layer
                lines = options.includeBBox ? textLayerLines(page) : nil
                // Direct pages only need pixels for the page image or for barcode regions.
                guard writesImage || options.includeBBox else { return .success(PreparedPage(text: text, lines: lines, image: nil)) }
            }
            guard let image = render(page, dpi: options.dpi, color: writesImage) else {
                return .failure(.renderFailed("Failed to render page"))
            }
            return .success(PreparedPage(text: text, lines: lines, image: image))
        }()

        var pageImage: PageImage?
        do {
            let page = try prepared.get()
            pageImage = try page.image.flatMap { try PageImageWriter.write($0, page: number, options: options) }

            if let text = page.text {
                var barcodes: [Barcode]?
                var warnings: [String] = []
                if options.includeBBox, let image = page.image {
                    do {
                        barcodes = try TextRecognizer.detectBarcodes(VNImageRequestHandler(cgImage: image), timeout: options.timeout)
                    } catch {
                        warnings.append("barcode detection failed: \(error.localizedDescription)")
                    }
                }
                let result = PageResult(
                    index: number,
                    method: .direct,
                    content: text.trimmingCharacters(in: .whitespacesAndNewlines),
                    confidence: 1.0,
                    bbox: page.lines,
                    barcodes: barcodes,
                    image: pageImage,
                    warnings: warnings
                )
                return (result, nil)
            }

            guard let rendered = page.image, let image = writesImage ? PageImageWriter.grayscale(rendered) : rendered else {
                throw OCRError.renderFailed("Failed to render page")
            }
            let ocr = try TextRecognizer.recognize(image, options: options)
            let result = PageResult(
                index: number,
                method: .ocr,
                content: ocr.content,
                confidence: ocr.confidence,
                bbox: ocr.lines,
                barcodes: ocr.barcodes,
                skewDegrees: ocr.skewDegrees,
                image: pageImage,
                warnings: ocr.warnings
            )
            return (result, nil)
        } catch {
            let failure = error as? OCRError ?? .recognitionFailed(error.localizedDescription)
            return (.failed(index: number, error: failure, image: pageImage), failure)
        }
    }

    /// Text-layer lines with their boxes in the displayed (crop box, rotated) page, normalized like Vision's.
    static func textLayerLines(_ page: PDFPage) -> [BBox] {
        let box = PDFDisplayBox.cropBox
        let bounds = page.bounds(for: box)
        let quarterTurns = ((page.rotation % 360) + 360) % 360 / 90
        let size = quarterTurns.isMultiple(of: 2) ? bounds.size : CGSize(width: bounds.height, height: bounds.width)
        guard size.width > 0, size.height > 0, let selection = page.selection(for: bounds) else { return [] }
        // The same page-to-display transform `draw(with:to:)` applies when rendering.
        let transform = page.transform(for: box)

        func normalize(_ rect: CGRect) -> CGRect {
            let r = rect.applying(transform)
            return CGRect(x: r.minX / size.width, y: r.minY / size.height, width: r.width / size.width, height: r.height / size.height)
        }

        return selection.selectionsByLine().compactMap { line in
            guard let text = line.string?.trimmingCharacters(in: .whitespacesAndNewlines), !text.isEmpty else { return nil }
            let normalized = normalize(line.bounds(for: page))
            let words = textLayerWords(line, page: page, text: text).map { words in
                words.map { Word(text: $0.text, quad: Quad(normalize($0.bounds))) }
            }
            return BBox(text: text, confidence: 1.0, rect: normalized, quad: Quad(normalized), words: words)
        }
    }

    /// Words of a text-layer line with their page-space bounds (union of the characters' bounds). Nil when the
    /// line's characters cannot be matched to its text, e.g. ligatures that PDFKit reports as several characters.
    private static func textLayerWords(_ line: PDFSelection, page: PDFPage, text: String) -> [(text: String, bounds: CGRect)]? {
        var characters: [(character: String, bounds: CGRect)] = []
        for i in 0..<line.numberOfTextRanges(on: page) {
            let range = line.range(at: i, on: page)
            guard range.location != NSNotFound else { continue }
            // One selection per character: PDFPage.characterBounds(at:) and page.string do not share the selection
            // index space (they drift by one per line) and ignore the page rotation. PDFKit snaps the selection of a
            // line break to the previous character (and that character's selection takes the break along), so a
            // selection starting elsewhere counts as a space and line breaks are dropped from the string.
            for index in range.location..<(range.location + range.length) {
                guard let character = page.selection(for: NSRange(location: index, length: 1)),
                      character.numberOfTextRanges(on: page) >= 1,
                      character.range(at: 0, on: page).location == index
                else {
                    characters.append((" ", .null))
                    continue
                }
                let string = (character.string ?? "").trimmingCharacters(in: .newlines)
                characters.append((string.isEmpty ? " " : string, character.bounds(for: page)))
            }
        }
        var words: [(text: String, bounds: CGRect)] = []
        var current = "", bounds = CGRect.null
        for (character, box) in characters {
            if character.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty {
                if !current.isEmpty { words.append((current, bounds)) }
                current = ""
                bounds = .null
            } else {
                current += character
                if !box.isEmpty { bounds = bounds.union(box) }
            }
        }
        if !current.isEmpty { words.append((current, bounds)) }
        let expected = text.split(whereSeparator: \.isWhitespace).map(String.init)
        guard words.map(\.text) == expected, words.allSatisfy({ !$0.bounds.isNull }) else { return nil }
        return words
    }

    /// Renders the visible (crop box) area of `page` into an 8-bit grayscale (or RGB) bitmap, honoring page rotation.
    static func render(_ page: PDFPage, dpi: CGFloat, color: Bool = false) -> CGImage? {
        let box = PDFDisplayBox.cropBox
        let bounds = page.bounds(for: box)
        let quarterTurns = ((page.rotation % 360) + 360) % 360 / 90
        let size = quarterTurns.isMultiple(of: 2) ? bounds.size : CGSize(width: bounds.height, height: bounds.width)
        guard size.width > 0, size.height > 0 else { return nil }

        let scale = min(dpi / 72.0, maxRenderDimension / max(size.width, size.height))
        let width = Int((size.width * scale).rounded())
        let height = Int((size.height * scale).rounded())

        guard let context = CGContext(
            data: nil,
            width: width,
            height: height,
            bitsPerComponent: 8,
            bytesPerRow: 0,
            space: color ? CGColorSpace(name: CGColorSpace.sRGB)! : CGColorSpaceCreateDeviceGray(),
            bitmapInfo: color ? CGImageAlphaInfo.noneSkipLast.rawValue : CGImageAlphaInfo.none.rawValue
        ) else { return nil }

        context.interpolationQuality = .high
        context.setFillColor(CGColor(gray: 1, alpha: 1))
        context.fill(CGRect(x: 0, y: 0, width: width, height: height))
        context.scaleBy(x: scale, y: scale)
        // `draw(with:to:)` applies `transform(for:)` itself: box origin to zero plus page rotation.
        page.draw(with: box, to: context)
        return context.makeImage()
    }

    /// `text` is set for text-layer pages; `image` whenever pixels are needed.
    private struct PreparedPage {
        let text: String?
        let lines: [BBox]?
        let image: CGImage?
    }
}

/// Decides whether a PDF's embedded text layer is trustworthy enough to skip OCR.
enum TextLayer {
    static let minimumPrintable = 50
    static let minimumAlphanumericRatio = 0.40

    static func isUsable(_ text: String) -> Bool {
        var printable = 0
        var alphanumeric = 0
        for character in text {
            if character.isLetter || character.isNumber {
                alphanumeric += 1
                printable += 1
            } else if character.isWhitespace || character.isPunctuation {
                printable += 1
            }
        }
        guard printable >= minimumPrintable else { return false }
        return Double(alphanumeric) / Double(printable) >= minimumAlphanumericRatio
    }
}
