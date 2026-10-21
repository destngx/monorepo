import CoreImage
import Foundation
import Vision

/// Second look at empty table cells. Vision's page pass skips lone short tokens such as a "%" unit; recognizing
/// the cell alone, deskewed and enlarged, usually reads them, and finds nothing in cells that really are blank.
enum CellReader {
    static let scale: CGFloat = 3
    static let minimumConfidence = 0.5
    /// Bounds the extra work on pages with many blank cells.
    static let maximumCellsPerPage = 24

    struct Reading {
        let text: String
        let confidence: Double
        /// Normalized, bottom-left origin, in the upright page.
        let quad: Quad
        let rect: CGRect
        let words: [Word]?
    }

    /// `region` is in deskewed pixel space (the upright page rotated by `-skew` about its origin).
    static func read(
        _ upright: CIImage,
        region: CGRect,
        skew: Double,
        languages: [String],
        timeout: TimeInterval,
        context: CIContext
    ) throws -> Reading? {
        let size = upright.extent.size
        let pad = max(2, region.height * 0.15)
        let crop = region.insetBy(dx: -pad, dy: -pad)
        let deskewed = upright.transformed(by: CGAffineTransform(rotationAngle: -skew))
        // A white margin keeps glyphs off the crop edge, where Vision tends to drop them.
        let margin: CGFloat = 40
        let enlarged = deskewed
            .cropped(to: crop)
            .transformed(by: CGAffineTransform(translationX: -crop.minX, y: -crop.minY)
                .concatenating(CGAffineTransform(scaleX: scale, y: scale)))
        let canvas = CGRect(x: -margin, y: -margin, width: crop.width * scale + 2 * margin, height: crop.height * scale + 2 * margin)
        let flattened = enlarged.composited(over: CIImage(color: .white).cropped(to: canvas))
        guard let image = context.createCGImage(flattened, from: canvas, format: .L8, colorSpace: CGColorSpaceCreateDeviceGray()) else {
            throw OCRError.renderFailed("Failed to crop table cell")
        }

        let request = VNRecognizeTextRequest()
        request.recognitionLevel = .accurate
        request.usesLanguageCorrection = true
        request.recognitionLanguages = languages
        try TextRecognizer.perform([request], on: VNImageRequestHandler(cgImage: image), timeout: timeout)

        let candidates = (request.results ?? []).compactMap { observation in
            observation.topCandidates(1).first.map { (observation, $0) }
        }
        guard !candidates.isEmpty else { return nil }
        let text = candidates.map(\.1.string).joined(separator: " ").trimmingCharacters(in: .whitespaces)
        let confidence = candidates.map { Double($0.1.confidence) }.min()!
        guard !text.isEmpty, confidence >= minimumConfidence else { return nil }

        // Map back: canvas -> crop -> deskewed page -> upright page -> normalized.
        let union = candidates.map(\.0.boundingBox).reduce(CGRect.null) { $0.union($1) }
        let toPage = CGAffineTransform(rotationAngle: skew)
        func pagePoint(_ p: CGPoint) -> Point { point(p.x, p.y) }
        func point(_ x: CGFloat, _ y: CGFloat) -> Point {
            let cx = canvas.minX + x * canvas.width, cy = canvas.minY + y * canvas.height
            let p = CGPoint(x: crop.minX + cx / scale, y: crop.minY + cy / scale).applying(toPage)
            return Point(x: Double(p.x / size.width), y: Double(p.y / size.height))
        }
        let quad = Quad(
            topLeft: point(union.minX, union.maxY),
            topRight: point(union.maxX, union.maxY),
            bottomRight: point(union.maxX, union.minY),
            bottomLeft: point(union.minX, union.minY)
        )
        let corners = [quad.topLeft, quad.topRight, quad.bottomRight, quad.bottomLeft]
        let xs = corners.map(\.x), ys = corners.map(\.y)
        let rect = CGRect(x: xs.min()!, y: ys.min()!, width: xs.max()! - xs.min()!, height: ys.max()! - ys.min()!)
        // Words in the joined text, boxed by the reading they came from.
        var words: [Word]? = []
        boxing: for (_, candidate) in candidates {
            for range in TextRecognizer.tokenRanges(in: candidate.string) {
                guard let box = try? candidate.boundingBox(for: range) else {
                    words = nil
                    break boxing
                }
                words?.append(Word(text: String(candidate.string[range]), quad: Quad(
                    topLeft: pagePoint(box.topLeft), topRight: pagePoint(box.topRight),
                    bottomRight: pagePoint(box.bottomRight), bottomLeft: pagePoint(box.bottomLeft)
                )))
            }
        }
        return Reading(text: text, confidence: confidence, quad: quad, rect: rect, words: words)
    }
}
