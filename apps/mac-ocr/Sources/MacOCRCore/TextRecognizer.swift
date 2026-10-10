import Foundation
import Vision

struct RecognizedText {
    let content: String
    let confidence: Double
    let lines: [BBox]?
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

    /// Runs Vision text recognition on `handler`, cancelling the request once `options.timeout` elapses.
    static func recognize(_ handler: VNImageRequestHandler, options: OCROptions) throws -> RecognizedText {
        let request = VNRecognizeTextRequest()
        request.recognitionLevel = .accurate
        request.usesLanguageCorrection = true
        request.recognitionLanguages = options.languages

        let timedOut = TimeoutFlag()
        let watchdog = DispatchWorkItem {
            timedOut.set()
            request.cancel()
        }
        DispatchQueue.global(qos: .utility).asyncAfter(deadline: .now() + options.timeout, execute: watchdog)
        defer { watchdog.cancel() }

        do {
            try handler.perform([request])
        } catch {
            if timedOut.isSet {
                throw OCRError.timeout("Recognition exceeded \(formatSeconds(options.timeout))s timeout")
            }
            throw OCRError.recognitionFailed("Text recognition failed: \(error.localizedDescription)")
        }
        if timedOut.isSet {
            throw OCRError.timeout("Recognition exceeded \(formatSeconds(options.timeout))s timeout")
        }

        let observations = request.results ?? []
        var lines: [String] = []
        var boxes: [BBox] = []
        var totalConfidence = 0.0
        lines.reserveCapacity(observations.count)

        for observation in observations {
            guard let candidate = observation.topCandidates(1).first else { continue }
            let confidence = Double(candidate.confidence)
            lines.append(candidate.string)
            totalConfidence += confidence

            if options.includeBBox {
                let box = observation.boundingBox
                boxes.append(BBox(
                    text: candidate.string,
                    confidence: confidence,
                    x: Double(box.origin.x),
                    y: Double(box.origin.y),
                    w: Double(box.width),
                    h: Double(box.height)
                ))
            }
        }

        return RecognizedText(
            content: lines.joined(separator: "\n").trimmingCharacters(in: .whitespacesAndNewlines),
            confidence: lines.isEmpty ? 0 : totalConfidence / Double(lines.count),
            lines: options.includeBBox ? boxes : nil
        )
    }

    private static func formatSeconds(_ value: TimeInterval) -> String {
        value == value.rounded() ? String(Int(value)) : String(value)
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
