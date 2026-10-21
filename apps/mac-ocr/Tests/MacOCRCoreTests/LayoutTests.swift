import CoreGraphics
import XCTest
@testable import MacOCRCore

final class LayoutTests: XCTestCase {
    private let page = CGSize(width: 1000, height: 1400)

    /// A lab table delivered column by column, as Vision does for ruled tables.
    private var table: [(String, CGFloat, CGFloat)] {
        let rows: [[String]] = [
            ["STT", "Tên xét nghiệm", "Kết quả", "Đơn vị", "Tham chiếu"],
            ["1", "Glucose", "6.4 H", "mmol/L", "3.9 - 6.1"],
            ["2", "HbA1c", "6.1 H", "%", "4.0 - 5.6"],
            ["3", "Creatinin", "88", "µmol/L", "62 - 120"],
        ]
        let columns: [CGFloat] = [100, 200, 500, 650, 800]
        var lines: [(String, CGFloat, CGFloat)] = []
        for column in columns.indices {
            for (row, cells) in rows.enumerated() {
                lines.append((cells[column], columns[column], 1000 - CGFloat(row) * 40))
            }
        }
        return lines
    }

    func testTableReadsRowByRow() {
        XCTAssertEqual(layout(table), """
        STT\tTên xét nghiệm\tKết quả\tĐơn vị\tTham chiếu
        1\tGlucose\t6.4 H\tmmol/L\t3.9 - 6.1
        2\tHbA1c\t6.1 H\t%\t4.0 - 5.6
        3\tCreatinin\t88\tµmol/L\t62 - 120
        """)
    }

    func testTiltedTableStillReadsRowByRow() {
        for degrees in [-7.0, -2.5, 4.5, 7.0] {
            let angle = degrees * .pi / 180
            XCTAssertEqual(layout(table, rotation: angle), layout(table), "\(degrees)°")
            XCTAssertEqual(Layout.skew(lines(table, rotation: angle), imageSize: page), angle, accuracy: 0.25 * .pi / 180, "\(degrees)°")
        }
    }

    func testMissingCellIsLeftEmptyInItsColumn() {
        let withoutUnit = table.filter { $0.0 != "%" }
        XCTAssertTrue(layout(withoutUnit).contains("2\tHbA1c\t6.1 H\t\t4.0 - 5.6"))
    }

    func testWrappedProseInTwoColumnsStaysTogether() {
        let prose: [(String, CGFloat, CGFloat)] = [
            ("Chẩn đoán: Viêm phổi thùy dưới phải, theo", 60, 1000),
            ("dõi đái tháo đường type 2.", 60, 975),
            ("Điều trị: Kháng sinh đường tĩnh mạch bảy", 540, 1000),
            ("ngày, kiểm soát đường huyết bằng", 540, 975),
            ("insulin.", 540, 950),
            ("Lời dặn của bác sĩ:", 60, 900),
        ]
        XCTAssertEqual(layout(prose), """
        Chẩn đoán: Viêm phổi thùy dưới phải, theo dõi đái tháo đường type 2.\tĐiều trị: Kháng sinh đường tĩnh mạch bảy ngày, kiểm soát đường huyết bằng insulin.
        Lời dặn của bác sĩ:
        """)
    }

    func testLinesAreNotGluedAcrossSentencesOrShortCells() {
        let lines: [(String, CGFloat, CGFloat)] = [
            ("Họ tên người bệnh: NGUYỄN VĂN MẪU.", 60, 1000),
            ("dòng này bắt đầu câu mới sau dấu chấm", 60, 975),
            ("mmol/L", 650, 900),
            ("mmol/L", 650, 875),
        ]
        XCTAssertEqual(layout(lines), """
        Họ tên người bệnh: NGUYỄN VĂN MẪU.
        dòng này bắt đầu câu mới sau dấu chấm
        mmol/L
        mmol/L
        """)
    }

    func testRawKeepsVisionOrder() {
        let raw = Layout.text(lines(table), imageSize: page, mode: .raw)
        XCTAssertEqual(raw, table.map(\.0).joined(separator: "\n"))
    }

    func testEmptyInput() {
        XCTAssertEqual(Layout.text([], imageSize: page, mode: .rows), "")
        XCTAssertEqual(Layout.skew([], imageSize: page), 0)
    }

    // MARK: Helpers

    private func layout(_ items: [(String, CGFloat, CGFloat)], rotation: Double = 0) -> String {
        Layout.text(lines(items, rotation: rotation), imageSize: page, mode: .rows)
    }

    /// Lines in pixel space (bottom-left origin, 9 px per character, 20 px tall), rotated about the page centre.
    private func lines(_ items: [(String, CGFloat, CGFloat)], rotation: Double = 0) -> [LayoutLine] {
        let center = CGPoint(x: page.width / 2, y: page.height / 2)
        func point(_ x: CGFloat, _ y: CGFloat) -> Point {
            let dx = Double(x - center.x), dy = Double(y - center.y)
            let rx = Double(center.x) + dx * cos(rotation) - dy * sin(rotation)
            let ry = Double(center.y) + dx * sin(rotation) + dy * cos(rotation)
            return Point(x: rx / Double(page.width), y: ry / Double(page.height))
        }
        return items.map { text, x, y in
            let w = CGFloat(text.count * 9), h: CGFloat = 20
            return LayoutLine(text: text, quad: Quad(
                topLeft: point(x, y + h), topRight: point(x + w, y + h),
                bottomRight: point(x + w, y), bottomLeft: point(x, y)
            ))
        }
    }
}

final class LayoutAnalysisTests: XCTestCase {
    func testEmptyCellRegionCoversItsColumnInTheRowBand() {
        let page = CGSize(width: 1000, height: 1400)
        func line(_ text: String, _ x: Double, _ y: Double) -> LayoutLine {
            let w = Double(text.count * 9), h = 20.0
            func p(_ px: Double, _ py: Double) -> Point { Point(x: px / 1000, y: py / 1400) }
            return LayoutLine(text: text, quad: Quad(topLeft: p(x, y + h), topRight: p(x + w, y + h), bottomRight: p(x + w, y), bottomLeft: p(x, y)))
        }
        let lines = [
            line("1", 100, 1000), line("Glucose", 200, 1000), line("6.4", 450, 1000), line("mmol/L", 650, 1000),
            line("2", 100, 960), line("HbA1c", 200, 960), line("6.1", 450, 960),
            line("3", 100, 920), line("Ure", 200, 920), line("5.1", 450, 920), line("mmol/L", 650, 920),
        ]
        let analysis = Layout.analyze(lines, imageSize: page)
        XCTAssertEqual(analysis.text, "1\tGlucose\t6.4\tmmol/L\n2\tHbA1c\t6.1\t\n3\tUre\t5.1\tmmol/L")
        guard analysis.rows.count == 3, analysis.rows[1].count == 4 else { return XCTFail("unexpected rows \(analysis.rows)") }
        let empty = analysis.rows[1][3]
        XCTAssertEqual(empty.text, "")
        XCTAssertEqual(empty.region.minX, 650, accuracy: 0.5)
        XCTAssertEqual(empty.region.maxX, 650 + 54, accuracy: 0.5)
        XCTAssertEqual(empty.region.minY, 960, accuracy: 0.5)
        XCTAssertEqual(empty.region.maxY, 980, accuracy: 0.5)
    }
}
