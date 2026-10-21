import CoreImage
import Foundation
import Vision

struct RecognizedText {
    let content: String
    let confidence: Double
    let lines: [BBox]?
    let barcodes: [Barcode]?
    let skewDegrees: Double?
    /// Optional steps that did not finish; the page text is still valid.
    let warnings: [String]
}

enum TextRecognizer {
    static func supportedLanguages() throws -> [String] {
        try VNRecognizeTextRequest().supportedRecognitionLanguages()
    }

    /// Throws `.usage` listing the supported codes when any requested language is unknown to Vision.
    static func validate(languages: [String]) throws {
        let supported: [String]
        do {
            supported = try supportedLanguages()
        } catch {
            throw OCRError.recognitionFailed("Failed to query supported languages: \(error.localizedDescription)")
        }
        let unsupported = languages.filter { !supported.contains($0) }
        guard unsupported.isEmpty else {
            throw OCRError.usage(
                "Unsupported language(s): \(unsupported.joined(separator: ", ")). Supported: \(supported.joined(separator: ", "))"
            )
        }
    }

    /// Runs Vision text recognition on `image` shown with `orientation`, failing once `options.timeout` elapses.
    /// Optional steps share what is left of that budget and only add warnings when they fail: barcode detection
    /// (with `includeBBox`; `barcodes` is then nil) and, with the rows layout, a second look at empty table cells.
    static func recognize(_ image: CGImage, orientation: CGImagePropertyOrientation = .up, options: OCROptions) throws -> RecognizedText {
        let handler = VNImageRequestHandler(cgImage: image, orientation: orientation)
        let imageSize = orientation.swapsAxes
            ? CGSize(width: image.height, height: image.width)
            : CGSize(width: image.width, height: image.height)
        let request = VNRecognizeTextRequest()
        request.recognitionLevel = .accurate
        request.usesLanguageCorrection = true
        request.recognitionLanguages = options.languages
        let deadline = Date().addingTimeInterval(options.timeout)

        try perform([request], on: handler, timeout: options.timeout)
        var warnings: [String] = []

        let observations = request.results ?? []
        var lines: [LayoutLine] = []
        var boxes: [BBox] = []
        var totalConfidence = 0.0
        lines.reserveCapacity(observations.count)

        for observation in observations {
            guard let candidate = observation.topCandidates(1).first else { continue }
            let confidence = Double(candidate.confidence)
            let quad = Quad(observation)
            lines.append(LayoutLine(text: candidate.string, quad: quad))
            totalConfidence += confidence
            if options.includeBBox {
                boxes.append(BBox(
                    text: candidate.string, confidence: confidence, rect: observation.boundingBox, quad: quad,
                    words: words(of: candidate)
                ))
            }
        }

        let content: String
        let skew: Double
        switch options.layout {
        case .raw:
            content = Layout.text(lines, imageSize: imageSize, mode: .raw)
            skew = options.includeBBox ? Layout.skew(lines, imageSize: imageSize) : 0
        case .rows:
            var analysis = Layout.analyze(lines, imageSize: imageSize)
            let found = fillEmptyCells(&analysis, image: image, orientation: orientation, deadline: deadline, options: options, warnings: &warnings)
            for reading in found where options.includeBBox {
                boxes.append(BBox(
                    text: reading.text, confidence: reading.confidence, rect: reading.rect, quad: reading.quad,
                    words: reading.words
                ))
            }
            content = analysis.text
            skew = analysis.skew
        }

        var barcodes: [Barcode]?
        if options.includeBBox {
            do {
                barcodes = try detectBarcodes(handler, timeout: deadline.timeIntervalSinceNow)
            } catch {
                warnings.append("barcode detection failed: \(error.localizedDescription)")
            }
        }

        return RecognizedText(
            content: trimmed(content, layout: options.layout),
            confidence: lines.isEmpty ? 0 : totalConfidence / Double(lines.count),
            lines: options.includeBBox ? boxes : nil,
            barcodes: barcodes,
            skewDegrees: options.includeBBox ? skew * 180 / .pi : nil,
            warnings: warnings
        )
    }

    /// Each whitespace-separated token with Vision's box for it; nil if Vision cannot box one of them.
    static func words(of candidate: VNRecognizedText) -> [Word]? {
        let string = candidate.string
        var words: [Word] = []
        for range in tokenRanges(in: string) {
            guard let box = try? candidate.boundingBox(for: range) else { return nil }
            words.append(Word(text: String(string[range]), quad: Quad(box)))
        }
        return words
    }

    static func tokenRanges(in string: String) -> [Range<String.Index>] {
        var ranges: [Range<String.Index>] = []
        var start: String.Index?
        for index in string.indices {
            if string[index].isWhitespace {
                if let s = start { ranges.append(s..<index) }
                start = nil
            } else if start == nil {
                start = index
            }
        }
        if let s = start { ranges.append(s..<string.endIndex) }
        return ranges
    }

    /// Rows keep leading and trailing tabs: they mark empty cells in the first and last rows.
    static func trimmed(_ content: String, layout: TextLayout) -> String {
        content.trimmingCharacters(in: layout == .rows ? .newlines : .whitespacesAndNewlines)
    }

    /// Stops at the first failure or once `deadline` passes, keeping what was read so far.
    static func fillEmptyCells(
        _ analysis: inout Layout.Analysis,
        image: CGImage,
        orientation: CGImagePropertyOrientation,
        deadline: Date,
        options: OCROptions,
        warnings: inout [String]
    ) -> [CellReader.Reading] {
        let empty = analysis.rows.indices.flatMap { r in analysis.rows[r].indices.filter { analysis.rows[r][$0].text.isEmpty }.map { (r, $0) } }
        guard !empty.isEmpty else { return [] }
        let upright = CIImage(cgImage: image).oriented(orientation)
        let page = upright.transformed(by: CGAffineTransform(translationX: -upright.extent.minX, y: -upright.extent.minY))
        let context = CIContext(options: [.useSoftwareRenderer: false])
        var found: [CellReader.Reading] = []
        for (r, c) in empty.prefix(CellReader.maximumCellsPerPage) {
            do {
                let remaining = deadline.timeIntervalSinceNow
                guard remaining > 0 else { throw OCRError.timeout("page timeout reached") }
                guard let reading = try CellReader.read(
                    page, region: analysis.rows[r][c].region, skew: analysis.skew,
                    languages: options.languages, timeout: remaining, context: context
                ) else { continue }
                analysis.rows[r][c].text = reading.text
                found.append(reading)
            } catch {
                warnings.append("empty table cells not re-read: \(error.localizedDescription)")
                break
            }
        }
        return found
    }

    static func detectBarcodes(_ handler: VNImageRequestHandler, timeout: TimeInterval) throws -> [Barcode] {
        guard timeout > 0 else { throw OCRError.timeout("page timeout reached") }
        let request = VNDetectBarcodesRequest()
        try perform([request], on: handler, timeout: timeout)
        return barcodes(from: request)
    }

    private static func barcodes(from request: VNDetectBarcodesRequest) -> [Barcode] {
        (request.results ?? []).map { observation in
            Barcode(
                symbology: observation.symbology.rawValue.replacingOccurrences(of: "VNBarcodeSymbology", with: ""),
                quad: Quad(observation)
            )
        }
    }

    static func perform(_ requests: [VNRequest], on handler: VNImageRequestHandler, timeout: TimeInterval) throws {
        let timedOut = TimeoutFlag()
        let watchdog = DispatchWorkItem {
            timedOut.set()
            requests.forEach { $0.cancel() }
        }
        DispatchQueue.global(qos: .utility).asyncAfter(deadline: .now() + timeout, execute: watchdog)
        defer { watchdog.cancel() }

        do {
            try handler.perform(requests)
        } catch {
            if timedOut.isSet {
                throw OCRError.timeout("Recognition exceeded \(formatSeconds(timeout))s timeout")
            }
            throw OCRError.recognitionFailed("Text recognition failed: \(error.localizedDescription)")
        }
        if timedOut.isSet {
            throw OCRError.timeout("Recognition exceeded \(formatSeconds(timeout))s timeout")
        }
    }

    private static func formatSeconds(_ value: TimeInterval) -> String {
        value == value.rounded() ? String(Int(value)) : String(value)
    }
}

extension Quad {
    init(_ observation: VNRectangleObservation) {
        func point(_ p: CGPoint) -> Point { Point(x: Double(p.x), y: Double(p.y)) }
        self.init(
            topLeft: point(observation.topLeft),
            topRight: point(observation.topRight),
            bottomRight: point(observation.bottomRight),
            bottomLeft: point(observation.bottomLeft)
        )
    }
}

private final class TimeoutFlag: @unchecked Sendable {
    private let lock = NSLock()
    private var value = false

    var isSet: Bool {
        lock.lock()
        defer { lock.unlock() }
        return value
    }

    func set() {
        lock.lock()
        value = true
        lock.unlock()
    }
}
