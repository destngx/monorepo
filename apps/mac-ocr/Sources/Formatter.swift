import Foundation

class Formatter {
    static func format(results: [OCRResult], format: OutputFormat) -> String {
        switch format {
        case .plain:
            return results.map { result in
                let header = "--- File: \(result.metadata.filename) ---"
                let content = result.pages.map { $0.content }.joined(separator: "\n\n")
                return "\(header)\n\(content)"
            }.joined(separator: "\n\n")
        case .json:
            let encoder = JSONEncoder()
            encoder.outputFormatting = [.prettyPrinted, .withoutEscapingSlashes]
            
            if results.count == 1 {
                if let data = try? encoder.encode(results[0]) {
                    return String(data: data, encoding: .utf8) ?? "{}"
                }
            } else {
                if let data = try? encoder.encode(results) {
                    return String(data: data, encoding: .utf8) ?? "[]"
                }
            }
            return "{}"
        }
    }
}
