import Foundation

class Formatter {
    static func format(pages: [PageResult], config: OCRConfig) -> String {
        switch config.format {
        case .plain:
            return pages.map { $0.content }.joined(separator: "\n\n")
        case .json:
            let summary = OCRSummary(
                totalChars: pages.reduce(0) { $0 + $1.charCount },
                pagesDirect: pages.filter { $0.method == .direct }.count,
                pagesOCR: pages.filter { $0.method == .ocr }.count,
                avgConfidence: pages.count > 0 ? pages.reduce(0.0) { $0 + $1.confidence } / Double(pages.count) : 0.0,
                warnings: []
            )
            
            let metadata = OCRMetadata(
                filename: config.inputPath.lastPathComponent,
                fileSize: (try? config.inputPath.resourceValues(forKeys: [.fileSizeKey]).fileSize).map { Int64($0) } ?? 0,
                pageCount: pages.count,
                processedAt: ISO8601DateFormatter().string(from: Date()),
                config: ConfigMetadata(
                    dpi: Double(config.dpi),
                    languages: config.languages,
                    forceOCR: config.forceOCR
                )
            )
            
            let result = OCRResult(
                metadata: metadata,
                pages: pages,
                summary: summary
            )
            
            let encoder = JSONEncoder()
            encoder.outputFormatting = [.prettyPrinted, .withoutEscapingSlashes]
            if let data = try? encoder.encode(result) {
                return String(data: data, encoding: .utf8) ?? ""
            }
            return "{}"
        }
    }
}
