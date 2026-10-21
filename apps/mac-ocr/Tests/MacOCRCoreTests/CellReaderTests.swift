import CoreImage
import CoreText
import XCTest
@testable import MacOCRCore

final class CellReaderTests: XCTestCase {
    private let size = CGSize(width: 1200, height: 800)
    /// Where the word sits before tilting, in pixels (bottom-left origin).
    private let word = CGRect(x: 700, y: 500, width: 110, height: 30)

    func testReadsTheCellAndMapsItBack() throws {
        for degrees in [0.0, 4.0] {
            let skew = degrees * .pi / 180
            let reading = try XCTUnwrap(try read(page(text: "mmol/L", skew: skew), region: word, skew: skew), "\(degrees)°")
            XCTAssertEqual(reading.text, "mmol/L", "\(degrees)°")
            XCTAssertGreaterThanOrEqual(reading.confidence, CellReader.minimumConfidence)

            // The reported corners sit on the tilted word in the upright page.
            let expected = word.applying(CGAffineTransform(rotationAngle: skew))
            let center = CGPoint(x: (reading.quad.topLeft.x + reading.quad.bottomRight.x) / 2 * size.width,
                                 y: (reading.quad.topLeft.y + reading.quad.bottomRight.y) / 2 * size.height)
            XCTAssertEqual(center.x, expected.midX, accuracy: 20, "\(degrees)°")
            XCTAssertEqual(center.y, expected.midY, accuracy: 20, "\(degrees)°")
            let angle = atan2((reading.quad.topRight.y - reading.quad.topLeft.y) * size.height,
                              (reading.quad.topRight.x - reading.quad.topLeft.x) * size.width)
            XCTAssertEqual(angle, skew, accuracy: 0.01, "\(degrees)°")

            let words = try XCTUnwrap(reading.words, "\(degrees)°")
            XCTAssertEqual(words.map(\.text), ["mmol/L"], "\(degrees)°")
            for (a, b) in zip([words[0].quad.topLeft, words[0].quad.bottomRight], [reading.quad.topLeft, reading.quad.bottomRight]) {
                XCTAssertEqual(a.x * size.width, b.x * size.width, accuracy: 6, "\(degrees)°")
                XCTAssertEqual(a.y * size.height, b.y * size.height, accuracy: 6, "\(degrees)°")
            }
        }
    }

    func testBlankCellReadsNothing() throws {
        XCTAssertNil(try read(page(text: "mmol/L", skew: 0), region: word.offsetBy(dx: 0, dy: -200), skew: 0))
    }

    func testFillsEmptyCellOfAnEXIFRotatedImage() throws {
        // Stored sideways, shown upright with EXIF .left; the cell region is in upright pixel space.
        let stored = Fixtures.rotatedClockwise(try XCTUnwrap(CIContext().createCGImage(page(text: "mmol/L", skew: 0), from: CGRect(origin: .zero, size: size))))
        var analysis = Layout.Analysis(rows: [[Layout.Cell(text: "", region: word)]], skew: 0)
        var warnings: [String] = []
        let found = TextRecognizer.fillEmptyCells(
            &analysis, image: stored, orientation: .left, deadline: Date().addingTimeInterval(10), options: OCROptions(), warnings: &warnings
        )
        XCTAssertEqual(analysis.rows[0][0].text, "mmol/L")
        XCTAssertEqual(found.count, 1)
        XCTAssertEqual(warnings, [])
    }

    func testPastDeadlineKeepsTheTextAndWarns() throws {
        let image = try XCTUnwrap(CIContext().createCGImage(page(text: "mmol/L", skew: 0), from: CGRect(origin: .zero, size: size)))
        var analysis = Layout.Analysis(rows: [[Layout.Cell(text: "6.4", region: .zero), Layout.Cell(text: "", region: word)]], skew: 0)
        var warnings: [String] = []
        let found = TextRecognizer.fillEmptyCells(
            &analysis, image: image, orientation: .up, deadline: Date().addingTimeInterval(-1), options: OCROptions(), warnings: &warnings
        )
        XCTAssertEqual(found.count, 0)
        XCTAssertEqual(analysis.text, "6.4\t")
        XCTAssertEqual(warnings.count, 1)
    }

    // MARK: Helpers

    private func read(_ image: CIImage, region: CGRect, skew: Double) throws -> CellReader.Reading? {
        try CellReader.read(image, region: region, skew: skew, languages: ["en-US"], timeout: 10, context: CIContext())
    }

    /// A white page with `text` drawn in `word`, the whole page then rotated by `skew` about the origin.
    private func page(text: String, skew: Double) -> CIImage {
        let context = CGContext(
            data: nil, width: Int(size.width), height: Int(size.height), bitsPerComponent: 8, bytesPerRow: 0,
            space: CGColorSpaceCreateDeviceGray(), bitmapInfo: CGImageAlphaInfo.none.rawValue
        )!
        context.setFillColor(gray: 1, alpha: 1)
        context.fill(CGRect(origin: .zero, size: size))
        context.rotate(by: skew)
        let font = CTFontCreateWithName("Helvetica" as CFString, 26, nil)
        let line = CTLineCreateWithAttributedString(CFAttributedStringCreate(nil, text as CFString, [kCTFontAttributeName: font] as CFDictionary)!)
        context.textPosition = CGPoint(x: word.minX + 4, y: word.minY + 6)
        CTLineDraw(line, context)
        return CIImage(cgImage: context.makeImage()!)
    }
}
