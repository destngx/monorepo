import CoreGraphics
import Foundation

/// How OCR lines are assembled into page `content`.
public enum TextLayout: String, CaseIterable, Codable {
    /// One line per visual row: tables read row by row with tab-separated cells, wrapped prose is rejoined.
    case rows
    /// Vision's own order, one observation per line (reads ruled tables column by column).
    case raw
}

/// A recognized line with its corners, normalized with a bottom-left origin.
struct LayoutLine {
    let text: String
    let quad: Quad
}

/// Rebuilds reading order from line geometry. Works in pixel space so angles are true angles.
enum Layout {
    /// Skews beyond this are not searched; Vision itself reads poorly past it.
    static let maxSkewDegrees = 15.0

    static func text(_ lines: [LayoutLine], imageSize: CGSize, mode: TextLayout) -> String {
        switch mode {
        case .raw:
            return lines.map(\.text).joined(separator: "\n")
        case .rows:
            return analyze(lines, imageSize: imageSize).text
        }
    }

    /// Row-ordered cells. An empty `text` is a table cell nothing was read in; its `region` says where to look.
    struct Cell: Equatable {
        var text: String
        /// Deskewed pixel space: the page rotated by `-skew` about the image origin.
        let region: CGRect
    }

    struct Analysis {
        var rows: [[Cell]]
        /// Radians; see `skew(_:imageSize:)`.
        let skew: Double

        var text: String {
            rows.map { $0.map(\.text).joined(separator: "\t") }.joined(separator: "\n")
        }
    }

    static func analyze(_ lines: [LayoutLine], imageSize: CGSize) -> Analysis {
        let angle = skew(lines, imageSize: imageSize)
        return Analysis(rows: cells(of: rows(of: glued(boxes(lines, imageSize: imageSize, skew: angle)))), skew: angle)
    }

    /// Page skew in radians (positive: text rises left to right).
    ///
    /// A projection profile picks the coarse angle whose rotated line centres pile into the sharpest rows;
    /// the dx-weighted median angle between horizontal neighbours on those rows then refines it.
    static func skew(_ lines: [LayoutLine], imageSize: CGSize) -> Double {
        let shapes = lines.map { Shape($0.quad, imageSize) }
        guard !shapes.isEmpty else { return 0 }
        let lineHeight = median(shapes.map(\.height))
        guard lineHeight > 0 else { return 0 }

        // Points every line height along each line's centre, so a long line counts once per row it crosses.
        var samples: [CGPoint] = []
        for shape in shapes {
            let steps = max(1, Int(distance(shape.leftMid, shape.rightMid) / lineHeight))
            for step in 0...steps {
                let f = Double(step) / Double(steps)
                samples.append(CGPoint(
                    x: shape.leftMid.x + (shape.rightMid.x - shape.leftMid.x) * f,
                    y: shape.leftMid.y + (shape.rightMid.y - shape.leftMid.y) * f
                ))
            }
        }
        func sharpness(_ angle: Double) -> Double {
            var bins: [Int: Double] = [:]
            for p in samples {
                let y = -Double(p.x) * sin(angle) + Double(p.y) * cos(angle)
                bins[Int((y / (lineHeight / 2)).rounded(.down)), default: 0] += 1
            }
            return bins.values.reduce(0) { $0 + $1 * $1 }
        }
        let stepsPerDegree = 4.0
        let limit = Int(maxSkewDegrees * stepsPerDegree)
        let coarse = (-limit...limit)
            .map { Double($0) / stepsPerDegree * .pi / 180 }
            .max { sharpness($0) < sharpness($1) } ?? 0

        func rowY(_ shape: Shape) -> Double { -Double(shape.center.x) * sin(coarse) + Double(shape.center.y) * cos(coarse) }
        var angles: [(angle: Double, weight: Double)] = []
        for a in shapes {
            let neighbour = shapes
                .filter { b in
                    let tallest = max(a.height, b.height)
                    return b.minX > a.maxX && abs(rowY(b) - rowY(a)) < 0.5 * tallest && abs(b.height - a.height) < 0.5 * tallest
                }
                .min { $0.minX < $1.minX }
            guard let b = neighbour else { continue }
            let dx = Double(b.center.x - a.center.x)
            angles.append((atan2(Double(b.center.y - a.center.y), dx), dx))
        }
        angles.sort { $0.angle < $1.angle }
        let half = angles.reduce(0) { $0 + $1.weight } / 2
        var accumulated = 0.0
        for entry in angles {
            accumulated += entry.weight
            if accumulated >= half { return entry.angle }
        }
        return coarse
    }

    // MARK: Steps

    /// An axis-aligned box in deskewed pixel space (y up).
    struct Box {
        var text: String
        var minX: Double
        var maxX: Double
        var minY: Double
        var maxY: Double
        /// Height of one text line; `maxY - minY` grows when wrapped lines are glued.
        let lineHeight: Double
        /// Width of the last glued line, which decides whether the next line can continue it.
        var lastLineWidth: Double

        var height: Double { maxY - minY }
        var centerY: Double { (minY + maxY) / 2 }
    }

    private static func boxes(_ lines: [LayoutLine], imageSize: CGSize, skew angle: Double) -> [Box] {
        let (c, s) = (cos(-angle), sin(-angle))
        return lines.map { line in
            let corners = Shape.pixels(line.quad, imageSize).map { p in
                (x: Double(p.x) * c - Double(p.y) * s, y: Double(p.x) * s + Double(p.y) * c)
            }
            let xs = corners.map(\.x), ys = corners.map(\.y)
            return Box(
                text: line.text, minX: xs.min()!, maxX: xs.max()!, minY: ys.min()!, maxY: ys.max()!,
                lineHeight: ys.max()! - ys.min()!, lastLineWidth: xs.max()! - xs.min()!
            )
        }
        .sorted { $0.centerY > $1.centerY }
    }

    /// Rejoins wrapped prose: a line continues the one above when it starts lowercase and sits tight under a long
    /// line that does not end a sentence, with the same left edge and text height. Short cells (units, values)
    /// never continue, so stacked table columns stay apart.
    private static func glued(_ boxes: [Box]) -> [Box] {
        var used = Set<Int>()
        var result: [Box] = []
        for i in boxes.indices where !used.contains(i) {
            var paragraph = boxes[i]
            used.insert(i)
            while let j = boxes.indices.first(where: { !used.contains($0) && continues(paragraph, with: boxes[$0]) }) {
                let next = boxes[j]
                used.insert(j)
                paragraph.text += " " + next.text
                paragraph.minX = min(paragraph.minX, next.minX)
                paragraph.maxX = max(paragraph.maxX, next.maxX)
                paragraph.minY = next.minY
                paragraph.lastLineWidth = next.maxX - next.minX
            }
            result.append(paragraph)
        }
        return result
    }

    private static func continues(_ above: Box, with below: Box) -> Bool {
        let h = above.lineHeight
        let gap = above.minY - below.maxY
        guard gap > -0.4 * h, gap < 0.6 * h,
              abs(below.minX - above.minX) < 0.6 * h,
              abs(below.lineHeight - h) < 0.4 * h,
              above.lastLineWidth >= 8 * h,
              let first = below.text.first, first.isLetter, first.isLowercase,
              let last = above.text.last, !".:;!?".contains(last)
        else { return false }
        return true
    }

    /// Groups boxes (sorted top to bottom) into rows: a box joins the row whose band it overlaps most, as long as
    /// no box already in that row overlaps it horizontally. Rows are returned top to bottom, cells left to right.
    private static func rows(of boxes: [Box]) -> [[Box]] {
        var rows: [[Box]] = []
        for box in boxes {
            var best: (index: Int, overlap: Double)?
            for (index, row) in rows.enumerated() {
                if row.contains(where: { min($0.maxX, box.maxX) > max($0.minX, box.minX) }) { continue }
                let center = row.reduce(0.0) { $0 + $1.centerY } / Double(row.count)
                let height = median(row.map(\.height))
                let overlap = min(center + height / 2, box.maxY) - max(center - height / 2, box.minY)
                if overlap >= 0.5 * min(box.height, height), overlap > (best?.overlap ?? 0) {
                    best = (index, overlap)
                }
            }
            if let best { rows[best.index].append(box) } else { rows.append([box]) }
        }
        func center(_ row: [Box]) -> Double { row.reduce(0.0) { $0 + $1.centerY } / Double(row.count) }
        return rows
            .map { $0.sorted { $0.minX < $1.minX } }
            .sorted { center($0) > center($1) }
    }

    /// Cells per row. In a table block (consecutive rows of 3+ cells) the rows with the most cells define the
    /// column spans; a shorter row puts each cell in the span it overlaps most and leaves the others empty.
    private static func cells(of rows: [[Box]]) -> [[Cell]] {
        var cells = rows.map { $0.map { Cell(text: $0.text, region: CGRect(x: $0.minX, y: $0.minY, width: $0.maxX - $0.minX, height: $0.height)) } }
        var start = 0
        while start < rows.count {
            guard rows[start].count >= 3 else { start += 1; continue }
            var end = start
            while end < rows.count, rows[end].count >= 3 { end += 1 }
            defer { start = end }

            let block = rows[start..<end]
            let width = block.map(\.count).max()!
            let full = block.filter { $0.count == width }
            guard full.count >= 2 else { continue }
            let spans = (0..<width).map { column in
                (min: full.map { $0[column].minX }.min()!, max: full.map { $0[column].maxX }.max()!)
            }
            for index in start..<end where rows[index].count < width {
                if let placed = place(rows[index], in: spans) { cells[index] = placed }
            }
        }
        return cells
    }

    /// Nil when two cells would land in the same column or out of order; the row is then left as read.
    /// Empty cells span their column across the row's band.
    private static func place(_ row: [Box], in spans: [(min: Double, max: Double)]) -> [Cell]? {
        let bottom = row.map(\.minY).min()!, top = row.map(\.maxY).max()!
        var placed = spans.map { Cell(text: "", region: CGRect(x: $0.min, y: bottom, width: $0.max - $0.min, height: top - bottom)) }
        var previous = -1
        for box in row {
            func overlap(_ column: Int) -> Double { min(spans[column].max, box.maxX) - max(spans[column].min, box.minX) }
            let column = spans.indices.max { overlap($0) < overlap($1) }!
            guard column > previous else { return nil }
            placed[column] = Cell(text: box.text, region: CGRect(x: box.minX, y: box.minY, width: box.maxX - box.minX, height: box.height))
            previous = column
        }
        return placed
    }

    // MARK: Geometry

    /// A line's corners in pixel space.
    private struct Shape {
        let leftMid: CGPoint
        let rightMid: CGPoint
        let center: CGPoint
        let minX: CGFloat
        let maxX: CGFloat
        let height: Double

        init(_ quad: Quad, _ size: CGSize) {
            let p = Shape.pixels(quad, size)
            let (tl, tr, br, bl) = (p[0], p[1], p[2], p[3])
            leftMid = CGPoint(x: (tl.x + bl.x) / 2, y: (tl.y + bl.y) / 2)
            rightMid = CGPoint(x: (tr.x + br.x) / 2, y: (tr.y + br.y) / 2)
            center = CGPoint(x: (leftMid.x + rightMid.x) / 2, y: (leftMid.y + rightMid.y) / 2)
            minX = p.map(\.x).min()!
            maxX = p.map(\.x).max()!
            height = (distance(tl, bl) + distance(tr, br)) / 2
        }

        /// Top-left, top-right, bottom-right, bottom-left.
        static func pixels(_ quad: Quad, _ size: CGSize) -> [CGPoint] {
            [quad.topLeft, quad.topRight, quad.bottomRight, quad.bottomLeft].map {
                CGPoint(x: $0.x * Double(size.width), y: $0.y * Double(size.height))
            }
        }
    }

    private static func distance(_ a: CGPoint, _ b: CGPoint) -> Double {
        hypot(Double(b.x - a.x), Double(b.y - a.y))
    }

    private static func median(_ values: [Double]) -> Double {
        guard !values.isEmpty else { return 0 }
        return values.sorted()[values.count / 2]
    }
}
