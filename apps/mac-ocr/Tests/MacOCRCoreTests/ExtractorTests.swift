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

    private func assertReadsSentence(_ content: String, file: StaticString = #filePath, line: UInt = #line) {
        XCTAssertTrue(
            content.lowercased().contains(Fixtures.sentence.lowercased()),
            "expected OCR text to contain the fixture sentence, got: \(content)",
            file: file,
            line: line
        )
    }
}
