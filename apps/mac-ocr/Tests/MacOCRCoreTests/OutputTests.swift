import XCTest
@testable import MacOCRCore

final class TextLayerTests: XCTestCase {
    func testAcceptsProse() {
        XCTAssertTrue(TextLayer.isUsable(String(repeating: "Quarterly revenue grew 12%. ", count: 3)))
    }

    func testRejectsShortText() {
        XCTAssertFalse(TextLayer.isUsable("Page 1"))
    }

    func testRejectsMostlyPunctuation() {
        XCTAssertFalse(TextLayer.isUsable(String(repeating: ".-. ", count: 30) + "ab"))
    }

    func testIgnoresUnprintableGarbage() {
        // Control characters from broken font encodings do not count toward the threshold.
        XCTAssertFalse(TextLayer.isUsable(String(repeating: "\u{1}\u{2}\u{3}", count: 40) + "abc"))
    }
}

final class OutputFormatterTests: XCTestCase {
    private let url = URL(fileURLWithPath: "/nonexistent/doc.pdf")

    private func result(_ pages: [PageResult]) -> OCRResult {
        OCRResult.make(url: url, pages: pages, options: OCROptions(), processedAt: Date(timeIntervalSince1970: 0))
    }

    private let ok = PageResult(index: 1, method: .direct, content: "hello", confidence: 1, bbox: nil)
    private let failed = PageResult.failed(index: 2, error: .renderFailed("Failed to render page"))

    func testSummaryReportsFailedPagesAsWarnings() {
        let summary = result([ok, failed]).summary
        XCTAssertEqual(summary.totalChars, 5)
        XCTAssertEqual(summary.pagesDirect, 1)
        XCTAssertEqual(summary.pagesOCR, 1)
        XCTAssertEqual(summary.avgConfidence, 0.5)
        XCTAssertEqual(summary.warnings, ["page 2: Failed to render page"])
    }

    func testJSONShapeForSingleAndMultipleFiles() throws {
        let single = try JSONSerialization.jsonObject(with: Data(OutputFormatter.format([result([ok, failed])], as: .json).utf8))
        let object = try XCTUnwrap(single as? [String: Any])
        XCTAssertEqual(object["schema_version"] as? String, MacOCR.schemaVersion)
        let pages = try XCTUnwrap(object["pages"] as? [[String: Any]])
        XCTAssertEqual(pages[0]["char_count"] as? Int, 5)
        XCTAssertNil(pages[0]["error"])
        XCTAssertEqual(pages[1]["error"] as? String, "Failed to render page")
        XCTAssertEqual(pages[1]["content"] as? String, "")

        let multiple = try JSONSerialization.jsonObject(with: Data(OutputFormatter.format([result([ok]), result([ok])], as: .json).utf8))
        XCTAssertEqual((multiple as? [Any])?.count, 2)
    }

    func testPlainSkipsEmptyPages() throws {
        let output = try OutputFormatter.format([result([ok, failed])], as: .plain)
        XCTAssertEqual(output, "--- File: doc.pdf ---\nhello")
    }
}
