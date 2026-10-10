import Foundation

public enum OutputFormatter {
    /// JSON emits a single object for one file and an array for several (kept for backward compatibility).
    public static func format(_ results: [OCRResult], as format: OutputFormat) throws -> String {
        switch format {
        case .plain:
            return results.map { result in
                let content = result.pages.map(\.content).filter { !$0.isEmpty }.joined(separator: "\n\n")
                return "--- File: \(result.metadata.filename) ---\n\(content)"
            }.joined(separator: "\n\n")
        case .json:
            let encoder = JSONEncoder()
            encoder.outputFormatting = [.prettyPrinted, .withoutEscapingSlashes]
            let data = results.count == 1 ? try encoder.encode(results[0]) : try encoder.encode(results)
            return String(decoding: data, as: UTF8.self)
        }
    }
}
