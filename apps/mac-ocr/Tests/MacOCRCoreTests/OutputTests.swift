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

    func testSchema12Shapes() throws {
        let quad = Quad(CGRect(x: 0.1, y: 0.2, width: 0.3, height: 0.05))
        let page = PageResult(
            index: 1, method: .ocr, content: "a", confidence: 0.9,
            bbox: [BBox(text: "a", confidence: 0.9, rect: CGRect(x: 0.1, y: 0.2, width: 0.3, height: 0.05), quad: quad)],
            barcodes: [Barcode(symbology: "QR", quad: quad)],
            skewDegrees: 1.5,
            image: PageImage(path: "/tmp/pages/page-1.jpg", width: 10, height: 20)
        )
        let json = try OutputFormatter.format([result([page])], as: .json)
        let object = try XCTUnwrap(try JSONSerialization.jsonObject(with: Data(json.utf8)) as? [String: Any])
        XCTAssertEqual(object["schema_version"] as? String, "1.2")
        let encoded = try XCTUnwrap((object["pages"] as? [[String: Any]])?.first)

        let expectedQuad: [[Double]] = [[0.1, 0.25], [0.4, 0.25], [0.4, 0.2], [0.1, 0.2]]
        let box = try XCTUnwrap((encoded["bbox"] as? [[String: Any]])?.first)
        XCTAssertEqual(Set(box.keys), ["text", "confidence", "x", "y", "w", "h", "quad"])
        assertQuad(box["quad"], expectedQuad)
        let barcode = try XCTUnwrap((encoded["barcodes"] as? [[String: Any]])?.first)
        XCTAssertEqual(Set(barcode.keys), ["symbology", "quad"])
        assertQuad(barcode["quad"], expectedQuad)
        XCTAssertEqual(encoded["image"] as? [String: AnyHashable], ["path": "/tmp/pages/page-1.jpg", "width": 10, "height": 20])
        XCTAssertEqual(encoded["skew_degrees"] as? Double, 1.5)

        let decoded = try JSONDecoder().decode(OCRResult.self, from: Data(json.utf8))
        XCTAssertEqual(decoded.pages, [page])
    }

    func testOptionalFieldsAreOmittedWhenUnset() throws {
        let json = try OutputFormatter.format([result([ok])], as: .json)
        let page = try XCTUnwrap(((try JSONSerialization.jsonObject(with: Data(json.utf8)) as? [String: Any])?["pages"] as? [[String: Any]])?.first)
        XCTAssertEqual(Set(page.keys), ["index", "method", "content", "confidence", "char_count"])
    }

    private func assertQuad(_ value: Any?, _ expected: [[Double]], file: StaticString = #filePath, line: UInt = #line) {
        guard let quad = value as? [[Double]], quad.count == 4 else { return XCTFail("bad quad \(String(describing: value))", file: file, line: line) }
        for (got, want) in zip(quad, expected) {
            XCTAssertEqual(got[0], want[0], accuracy: 1e-9, file: file, line: line)
            XCTAssertEqual(got[1], want[1], accuracy: 1e-9, file: file, line: line)
        }
    }

    func testFailedPageKeepsWrittenImageAndWarningsReachSummary() throws {
        let image = PageImage(path: "/tmp/pages/page-2.jpg", width: 10, height: 20)
        let failedWithImage = PageResult.failed(index: 2, error: .timeout("Recognition exceeded 1s timeout"), image: image)
        let warned = PageResult(index: 1, method: .ocr, content: "a", confidence: 1, bbox: nil, warnings: ["barcode detection failed: x"])

        let output = result([warned, failedWithImage])
        XCTAssertEqual(output.pages[1].image, image)
        XCTAssertEqual(output.summary.warnings, ["page 1: barcode detection failed: x", "page 2: Recognition exceeded 1s timeout"])
        XCTAssertNil(result([ok]).pages[0].warnings)
    }

    func testRowsContentKeepsTabsOfEmptyEdgeCells() {
        XCTAssertEqual(TextRecognizer.trimmed("\tKết quả\tĐơn vị\n1\t6.4\t\n", layout: .rows), "\tKết quả\tĐơn vị\n1\t6.4\t")
        XCTAssertEqual(TextRecognizer.trimmed("  a\n", layout: .raw), "a")
    }

    func testPlainSkipsEmptyPages() throws {
        let output = try OutputFormatter.format([result([ok, failed])], as: .plain)
        XCTAssertEqual(output, "--- File: doc.pdf ---\nhello")
    }
}
