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

        let prepared: Result<PreparedPage, OCRError> = {
            documentLock.lock()
            defer { documentLock.unlock() }
            guard let page = document.page(at: pageIndex) else {
                return .failure(.renderFailed("Failed to load page"))
            }
            if !options.forceOCR, let text = page.string, TextLayer.isUsable(text) {
                return .success(.direct(text))
            }
            guard let image = render(page, dpi: options.dpi) else {
                return .failure(.renderFailed("Failed to render page"))
            }
            return .success(.image(image))
        }()

        switch prepared {
        case .failure(let error):
            return (.failed(index: number, error: error), error)
        case .success(.direct(let text)):
            let page = PageResult(
                index: number,
                method: .direct,
                content: text.trimmingCharacters(in: .whitespacesAndNewlines),
                confidence: 1.0,
                bbox: nil
            )
            return (page, nil)
        case .success(.image(let image)):
            do {
                let ocr = try TextRecognizer.recognize(VNImageRequestHandler(cgImage: image), options: options)
                return (PageResult(index: number, method: .ocr, content: ocr.content, confidence: ocr.confidence, bbox: ocr.lines), nil)
            } catch {
                let failure = error as? OCRError ?? .recognitionFailed(error.localizedDescription)
                return (.failed(index: number, error: failure), failure)
            }
        }
    }

    /// Renders the visible (crop box) area of `page` into an 8-bit grayscale bitmap, honoring page rotation.
    static func render(_ page: PDFPage, dpi: CGFloat) -> CGImage? {
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
            space: CGColorSpaceCreateDeviceGray(),
            bitmapInfo: CGImageAlphaInfo.none.rawValue
        ) else { return nil }

        context.interpolationQuality = .high
        context.setFillColor(gray: 1, alpha: 1)
        context.fill(CGRect(x: 0, y: 0, width: width, height: height))
        context.scaleBy(x: scale, y: scale)
        // `draw(with:to:)` applies `transform(for:)` itself: box origin to zero plus page rotation.
        page.draw(with: box, to: context)
        return context.makeImage()
    }

    private enum PreparedPage {
        case direct(String)
        case image(CGImage)
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
