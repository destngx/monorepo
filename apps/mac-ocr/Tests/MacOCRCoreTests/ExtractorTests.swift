import PDFKit
import UniformTypeIdentifiers
import XCTest
@testable import MacOCRCore

final class ExtractorTests: XCTestCase {
    private var directory: URL!

    override func setUpWithError() throws {
        directory = try Fixtures.makeDirectory()
    }

    override func tearDownWithError() throws {
        try FileManager.default.removeItem(at: directory)
    }

    // MARK: PDF

    func testTextLayerIsExtractedDirectly() throws {
        let url = file("text.pdf")
        Fixtures.textPDF(at: url, pages: 2)

        let result = try extract(url)

        XCTAssertEqual(result.pages.map(\.method), [.direct, .direct])
        XCTAssertEqual(result.pages.map(\.index), [1, 2])
        XCTAssertTrue(result.pages[0].content.contains(Fixtures.sentence))
        XCTAssertEqual(result.summary.pagesDirect, 2)
        XCTAssertEqual(result.summary.totalChars, result.pages.reduce(0) { $0 + $1.content.count })
    }

    func testScannedPagesFallBackToOCRInPageOrder() throws {
        let url = file("scan.pdf")
        Fixtures.scannedPDF(at: url, pages: 3)

        let result = try extract(url)

        XCTAssertEqual(result.pages.map(\.method), [.ocr, .ocr, .ocr])
        XCTAssertEqual(result.pages.map(\.index), [1, 2, 3])
        for page in result.pages {
            assertReadsSentence(page.content)
            XCTAssertGreaterThan(page.confidence, 0.5)
            XCTAssertNil(page.error)
        }
        XCTAssertEqual(result.summary.warnings, [])
    }

    func testForceOCRIgnoresTextLayer() throws {
        let url = file("text.pdf")
        Fixtures.textPDF(at: url)

        var options = OCROptions()
        options.forceOCR = true
        let page = try extract(url, options).pages[0]

        XCTAssertEqual(page.method, .ocr)
        assertReadsSentence(page.content)
    }

    func testTargetPage() throws {
        let url = file("text.pdf")
        Fixtures.textPDF(at: url, pages: 3)

        var options = OCROptions()
        options.targetPage = 2
        XCTAssertEqual(try extract(url, options).pages.map(\.index), [2])

        options.targetPage = 4
        XCTAssertThrowsError(try extract(url, options)) { error in
            XCTAssertEqual((error as? OCRError)?.exitCode, 1)
        }
    }

    func testRotatedPageIsRenderedUpright() throws {
        let url = file("rotated.pdf")
        Fixtures.rotatedTextPDF(at: url)
        let page = try XCTUnwrap(PDFDocument(url: url)?.page(at: 0))

        let image = try XCTUnwrap(PDFPipeline.render(page, dpi: 72))
        XCTAssertEqual(image.width, Int(Fixtures.letter.width))
        XCTAssertEqual(image.height, Int(Fixtures.letter.height))

        var options = OCROptions()
        options.forceOCR = true
        options.includeBBox = true
        assertUpright(try extract(url, options).pages[0])
    }

    func testRenderUsesCropBox() throws {
        let url = file("cropped.pdf")
        Fixtures.croppedPDF(at: url)
        let page = try XCTUnwrap(PDFDocument(url: url)?.page(at: 0))

        let image = try XCTUnwrap(PDFPipeline.render(page, dpi: 144))
        XCTAssertEqual(image.width, Int(Fixtures.letter.width * 2))
        XCTAssertEqual(image.height, Int(Fixtures.letter.height))

        var options = OCROptions()
        options.forceOCR = true
        options.includeBBox = true
        assertUpright(try extract(url, options).pages[0])
    }

    func testTimeoutFailsWithExitCode6() throws {
        let url = file("scan.pdf")
        Fixtures.scannedPDF(at: url)

        var options = OCROptions()
        options.timeout = 0.001
        XCTAssertThrowsError(try extract(url, options)) { error in
            XCTAssertEqual((error as? OCRError)?.exitCode, 6)
        }
    }

    // MARK: Images

    func testImageIsRecognized() throws {
        let url = file("scan.png")
        Fixtures.writeImage(Fixtures.scanImage(), to: url, type: .png)

        var options = OCROptions()
        options.includeBBox = true
        let page = try extract(url, options).pages[0]

        XCTAssertEqual(page.method, .ocr)
        assertUpright(page)
        for box in try XCTUnwrap(page.bbox) {
            XCTAssertTrue((0...1).contains(box.x) && (0...1).contains(box.y))
            XCTAssertGreaterThan(box.confidence, 0)
        }
    }

    func testImageHonorsExifOrientation() throws {
        let url = file("rotated.jpg")
        let sideways = Fixtures.rotatedClockwise(Fixtures.scanImage())
        Fixtures.writeImage(sideways, to: url, type: .jpeg, orientation: .left)

        var options = OCROptions()
        options.includeBBox = true
        assertUpright(try extract(url, options).pages[0])
    }

    // MARK: Layout

    func testTableIsReadRowByRow() throws {
        let url = file("table.png")
        Fixtures.writeImage(Fixtures.tableImage(), to: url, type: .png)

        let content = try extract(url).pages[0].content
        for (name, value) in Fixtures.tableRows {
            XCTAssertTrue(content.contains("\(name)\t\(value)"), "row \(name) not on one line: \(content)")
        }

        var raw = OCROptions()
        raw.layout = .raw
        XCTAssertFalse(try extract(url, raw).pages[0].content.contains("\t"))
    }

    // MARK: Engine contract (schema 1.2)

    func testRenderDirWritesEveryPDFPageIncludingTextLayerPages() throws {
        let text = file("text.pdf")
        Fixtures.textPDF(at: text, pages: 2)
        let output = directory.appendingPathComponent("pages/text")

        var options = OCROptions()
        options.renderDirectory = output
        options.dpi = 144
        let pages = try extract(text, options).pages

        XCTAssertEqual(pages.map(\.method), [.direct, .direct])
        for page in pages {
            let image = try XCTUnwrap(page.image)
            XCTAssertEqual(image.path, output.appendingPathComponent("page-\(page.index).jpg").path)
            XCTAssertEqual(image.width, Int(Fixtures.letter.width * 2))
            XCTAssertEqual(image.height, Int(Fixtures.letter.height * 2))
            let info = try XCTUnwrap(Fixtures.imageInfo(at: URL(fileURLWithPath: image.path)))
            XCTAssertEqual(info.type, UTType.jpeg.identifier)
            XCTAssertEqual(info.colorModel, "RGB")
            XCTAssertEqual(info.profile, "sRGB IEC61966-2.1")
            XCTAssertEqual(info.depth, 8)
            XCTAssertEqual([info.width, info.height], [image.width, image.height])
        }
    }

    func testRenderedPDFPagesMatchTheirLineBoxes() throws {
        for (name, make) in [("rotated.pdf", Fixtures.rotatedTextPDF(at:)), ("cropped.pdf", Fixtures.croppedPDF(at:))] {
            let url = file(name)
            make(url)
            var options = OCROptions()
            options.renderDirectory = directory.appendingPathComponent("pages-\(name)")
            options.includeBBox = true
            let page = try extract(url, options).pages[0]
            assertBoxesSitOnText(page, image: URL(fileURLWithPath: try XCTUnwrap(page.image).path))
        }
    }

    func testRenderDirWritesRotatedPDFPageUpright() throws {
        let url = file("rotated.pdf")
        Fixtures.rotatedTextPDF(at: url)

        var options = OCROptions()
        options.renderDirectory = directory.appendingPathComponent("pages")
        options.dpi = 72
        let image = try XCTUnwrap(try extract(url, options).pages[0].image)
        XCTAssertEqual([image.width, image.height], [Int(Fixtures.letter.width), Int(Fixtures.letter.height)])
    }

    func testRenderDirWritesImageInputUprightAsJPEG() throws {
        let url = file("rotated.png")
        let sideways = Fixtures.rotatedClockwise(Fixtures.scanImage())
        Fixtures.writeImage(sideways, to: url, type: .png, orientation: .left)

        var options = OCROptions()
        options.renderDirectory = directory.appendingPathComponent("pages")
        options.includeBBox = true
        let page = try extract(url, options).pages[0]

        let image = try XCTUnwrap(page.image)
        XCTAssertEqual(image.path, directory.appendingPathComponent("pages/page-1.jpg").path)
        XCTAssertEqual([image.width, image.height], [sideways.height, sideways.width])
        let info = try XCTUnwrap(Fixtures.imageInfo(at: URL(fileURLWithPath: image.path)))
        XCTAssertEqual(info.type, UTType.jpeg.identifier)
        XCTAssertEqual(info.colorModel, "RGB")
        XCTAssertEqual(info.profile, "sRGB IEC61966-2.1")
        XCTAssertEqual(info.depth, 8)
        assertUpright(page)
        assertBoxesSitOnText(page, image: URL(fileURLWithPath: image.path))
    }

    func testRenderDirRejectsSeveralInputs() throws {
        let a = file("a.pdf"), b = file("b.pdf")
        Fixtures.textPDF(at: a)
        Fixtures.textPDF(at: b)

        var options = OCROptions()
        options.renderDirectory = directory.appendingPathComponent("pages")
        XCTAssertThrowsError(try Extractor.extract(urls: [a, b], options: options)) { error in
            XCTAssertEqual((error as? OCRError)?.exitCode, 1)
        }
    }

    func testNoPageImagesWithoutRenderDir() throws {
        let url = file("text.pdf")
        Fixtures.textPDF(at: url)
        XCTAssertNil(try extract(url).pages[0].image)
    }

    func testOCRLinesCarryQuads() throws {
        let url = file("scan.png")
        Fixtures.writeImage(Fixtures.scanImage(), to: url, type: .png)

        var options = OCROptions()
        options.includeBBox = true
        let page = try extract(url, options).pages[0]

        for box in try XCTUnwrap(page.bbox) {
            let quad = try XCTUnwrap(box.quad)
            XCTAssertEqual(quad.bottomLeft.x, box.x, accuracy: 0.01)
            XCTAssertEqual(quad.bottomLeft.y, box.y, accuracy: 0.01)
            XCTAssertEqual(quad.topRight.x, box.x + box.w, accuracy: 0.01)
            XCTAssertEqual(quad.topRight.y, box.y + box.h, accuracy: 0.01)
        }
        XCTAssertEqual(try XCTUnwrap(page.skewDegrees), 0, accuracy: 0.5)
    }

    /// Text-layer boxes must land where OCR finds the same lines, including on rotated and cropped pages.
    func testTextLayerLinesMatchOCRLines() throws {
        for (name, make) in [("text.pdf", Fixtures.textPDF(at:pages:)), ("rotated.pdf", { url, _ in Fixtures.rotatedTextPDF(at: url) }),
                             ("cropped.pdf", { url, _ in Fixtures.croppedPDF(at: url) })] as [(String, (URL, Int) -> Void)] {
            let url = file(name)
            make(url, 1)

            var options = OCROptions()
            options.includeBBox = true
            let direct = try extract(url, options).pages[0]
            options.forceOCR = true
            let ocr = try extract(url, options).pages[0]

            XCTAssertEqual(direct.method, .direct, name)
            assertUpright(direct, file: #filePath, line: #line)
            let directBoxes = try XCTUnwrap(direct.bbox), ocrBoxes = try XCTUnwrap(ocr.bbox)
            XCTAssertEqual(directBoxes.count, ocrBoxes.count, name)
            for (d, o) in zip(directBoxes, ocrBoxes) {
                XCTAssertEqual(d.confidence, 1, name)
                XCTAssertNotNil(d.quad, name)
                XCTAssertEqual(d.x, o.x, accuracy: 0.02, "\(name): \(d.text)")
                XCTAssertEqual(d.y + d.h / 2, o.y + o.h / 2, accuracy: 0.02, "\(name): \(d.text)")
            }
        }
    }

    func testOCRLinesCarryWordQuadsInsideTheLine() throws {
        let url = file("scan.png")
        Fixtures.writeImage(Fixtures.scanImage(), to: url, type: .png)

        var options = OCROptions()
        options.includeBBox = true
        for box in try XCTUnwrap(extract(url, options).pages[0].bbox) {
            assertWords(of: box)
        }
    }

    /// Text-layer word boxes must land where OCR boxes the same words, including on rotated and cropped pages.
    func testTextLayerWordsMatchOCRWords() throws {
        for (name, make) in [("text.pdf", { Fixtures.textPDF(at: $0) }), ("rotated.pdf", Fixtures.rotatedTextPDF(at:)),
                             ("cropped.pdf", Fixtures.croppedPDF(at:))] as [(String, (URL) -> Void)] {
            let url = file(name)
            make(url)

            var options = OCROptions()
            options.includeBBox = true
            let direct = try XCTUnwrap(extract(url, options).pages[0].bbox)
            options.forceOCR = true
            let ocr = try XCTUnwrap(extract(url, options).pages[0].bbox)

            for box in direct { assertWords(of: box) }
            let directWords = direct.flatMap { $0.words ?? [] }, ocrWords = ocr.flatMap { $0.words ?? [] }
            for word in ["quick", "brown", "lazy"] {
                let d = try XCTUnwrap(directWords.first { $0.text == word }, "\(name): \(word)")
                let o = try XCTUnwrap(ocrWords.first { $0.text.lowercased() == word }, "\(name): \(word)")
                XCTAssertEqual(center(d.quad).x, center(o.quad).x, accuracy: 0.01, "\(name): \(word)")
                XCTAssertEqual(center(d.quad).y, center(o.quad).y, accuracy: 0.01, "\(name): \(word)")
            }
        }
    }

    func testBarcodesAreReportedWithoutPayload() throws {
        let url = file("qr.png")
        let (image, rect) = Fixtures.scanWithQRCode(payload: "PLACEHOLDER-ID-0000")
        Fixtures.writeImage(image, to: url, type: .png)

        var options = OCROptions()
        options.includeBBox = true
        let page = try extract(url, options).pages[0]

        let barcode = try XCTUnwrap(page.barcodes?.first)
        XCTAssertEqual(page.barcodes?.count, 1)
        XCTAssertEqual(barcode.symbology, "QR")
        for corner in [barcode.quad.topLeft, barcode.quad.topRight, barcode.quad.bottomRight, barcode.quad.bottomLeft] {
            XCTAssertTrue(rect.insetBy(dx: -0.02, dy: -0.02).contains(CGPoint(x: corner.x, y: corner.y)), "\(corner) outside \(rect)")
        }
        let json = try OutputFormatter.format([try extract(url, options)], as: .json)
        XCTAssertFalse(json.contains("PLACEHOLDER-ID-0000"))
        XCTAssertFalse(page.content.contains("PLACEHOLDER-ID-0000"))
    }

    func testBarcodesKeyIsEmptyWhenDetectionFoundNothing() throws {
        let text = file("text.pdf"), scan = file("scan.pdf")
        Fixtures.textPDF(at: text)
        Fixtures.scannedPDF(at: scan)

        var options = OCROptions()
        options.includeBBox = true
        XCTAssertEqual(try extract(text, options).pages[0].barcodes, [])
        XCTAssertEqual(try extract(scan, options).pages[0].barcodes, [])
        XCTAssertNil(try extract(scan).pages[0].barcodes)
        XCTAssertNil(try extract(text).pages[0].barcodes)
    }

    // MARK: Input validation

    func testMissingFileFailsWithExitCode2() {
        XCTAssertThrowsError(try Extractor.extract(urls: [file("missing.pdf")], options: OCROptions())) { error in
            XCTAssertEqual((error as? OCRError)?.exitCode, 2)
        }
    }

    func testDirectoryIsRejectedAsMissingFile() {
        XCTAssertThrowsError(try Extractor.extract(urls: [directory], options: OCROptions())) { error in
            XCTAssertEqual((error as? OCRError)?.exitCode, 2)
        }
    }

    func testUnsupportedTypeFailsWithExitCode3() throws {
        let url = file("notes.txt")
        try "hello".write(to: url, atomically: true, encoding: .utf8)

        XCTAssertThrowsError(try extract(url)) { error in
            XCTAssertEqual((error as? OCRError)?.exitCode, 3)
        }
    }

    func testUnsupportedLanguageFailsWithExitCode1() throws {
        let url = file("text.pdf")
        Fixtures.textPDF(at: url)

        var options = OCROptions()
        options.languages = ["vi-VN"]
        XCTAssertThrowsError(try extract(url, options)) { error in
            XCTAssertEqual((error as? OCRError)?.exitCode, 1)
        }
    }

    func testMultipleFilesKeepInputOrder() throws {
        let pdf = file("text.pdf")
        let png = file("scan.png")
        Fixtures.textPDF(at: pdf)
        Fixtures.writeImage(Fixtures.scanImage(), to: png, type: .png)

        let results = try Extractor.extract(urls: [png, pdf], options: OCROptions())
        XCTAssertEqual(results.map(\.metadata.filename), ["scan.png", "text.pdf"])
    }

    // MARK: Helpers

    private func file(_ name: String) -> URL {
        directory.appendingPathComponent(name)
    }

    private func extract(_ url: URL, _ options: OCROptions = OCROptions()) throws -> OCRResult {
        let results = try Extractor.extract(urls: [url], options: options)
        XCTAssertEqual(results.count, 1)
        return results[0]
    }

    /// Vision reads sideways text too, so content alone cannot prove orientation. Upright output has
    /// six wide, horizontal line boxes ordered top to bottom (bottom-left origin: y decreases).
    private func assertUpright(_ page: PageResult, file: StaticString = #filePath, line: UInt = #line) {
        assertReadsSentence(page.content, file: file, line: line)
        guard let boxes = page.bbox, boxes.count == 6 else {
            return XCTFail("expected 6 line boxes, got \(page.bbox?.count ?? 0)", file: file, line: line)
        }
        for box in boxes {
            XCTAssertGreaterThan(box.w, box.h * 3, "line box is not horizontal: \(box)", file: file, line: line)
        }
        XCTAssertEqual(boxes.map(\.y), boxes.map(\.y).sorted(by: >), "lines are not ordered top to bottom", file: file, line: line)
    }

    /// Line boxes cover dark text pixels in the page image; the same boxes moved off the text cover white paper.
    private func assertBoxesSitOnText(_ page: PageResult, image: URL, file: StaticString = #filePath, line: UInt = #line) {
        guard let boxes = page.bbox, !boxes.isEmpty else { return XCTFail("no line boxes", file: file, line: line) }
        for box in boxes {
            let rect = CGRect(x: box.x, y: box.y, width: box.w, height: box.h)
            XCTAssertLessThan(Fixtures.meanGray(at: image, in: rect), 225, "no text under \(box.text)", file: file, line: line)
        }
        let lowest = boxes.map(\.y).min()!
        let below = CGRect(x: boxes[0].x, y: lowest - 0.08, width: boxes[0].w, height: 0.05)
        XCTAssertGreaterThan(Fixtures.meanGray(at: image, in: below), 250, "text where none should be", file: file, line: line)
    }

    /// Words are the line's whitespace-separated tokens in order, each boxed inside the line box.
    private func assertWords(of box: BBox, file: StaticString = #filePath, line: UInt = #line) {
        guard let words = box.words else { return XCTFail("no words for \(box.text)", file: file, line: line) }
        XCTAssertEqual(words.map(\.text), box.text.split(whereSeparator: \.isWhitespace).map(String.init), file: file, line: line)
        let slack = box.h * 0.5
        for word in words {
            let c = center(word.quad)
            XCTAssertTrue(box.x - slack <= c.x && c.x <= box.x + box.w + slack && box.y - slack <= c.y && c.y <= box.y + box.h + slack,
                          "\(word.text) at \(c) is outside \(box.text)", file: file, line: line)
        }
        let xs = words.map { center($0.quad).x }
        XCTAssertEqual(xs, xs.sorted(), "words of \(box.text) are not left to right", file: file, line: line)
    }

    private func center(_ quad: Quad) -> CGPoint {
        CGPoint(x: (quad.topLeft.x + quad.bottomRight.x) / 2, y: (quad.topLeft.y + quad.bottomRight.y) / 2)
    }

    private func assertReadsSentence(_ content: String, file: StaticString = #filePath, line: UInt = #line) {
        XCTAssertTrue(
            content.lowercased().contains(Fixtures.sentence.lowercased()),
            "expected OCR text to contain the fixture sentence, got: \(content)",
            file: file,
            line: line
        )
    }
}
