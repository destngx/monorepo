import Foundation
import CoreGraphics

enum ExtractionMethod: String, Codable {
    case direct
    case ocr
}

struct BBox: Codable {
    let text: String
    let x: Double
    let y: Double
    let w: Double
    let h: Double
}

struct PageResult: Codable {
    let index: Int
    let method: ExtractionMethod
    let content: String
    let confidence: Double
    let charCount: Int
    let bbox: [BBox]?
    
    enum CodingKeys: String, CodingKey {
        case index
        case method
        case content
        case confidence
        case charCount = "char_count"
        case bbox
    }
}

struct OCRConfig {
    let inputPath: URL
    let format: OutputFormat
    let languages: [String]
    let dpi: CGFloat
    let forceOCR: Bool
    let includeBBox: Bool
    let timeout: TimeInterval
    let targetPage: Int? // 1-indexed
}

enum OutputFormat: String {
    case plain
    case json
}

struct OCRResult: Codable {
    let schemaVersion: String = "1.0"
    let metadata: OCRMetadata
    let pages: [PageResult]
    let summary: OCRSummary
    
    enum CodingKeys: String, CodingKey {
        case schemaVersion = "schema_version"
        case metadata
        case pages
        case summary
    }
}

struct OCRMetadata: Codable {
    let filename: String
    let fileSize : Int64
    let pageCount: Int
    let processedAt: String
    let toolVersion: String = "1.0.0"
    let config: ConfigMetadata
    
    enum CodingKeys: String, CodingKey {
        case filename
        case fileSize = "file_size_bytes"
        case pageCount = "page_count"
        case processedAt = "processed_at"
        case toolVersion = "tool_version"
        case config
    }
}

struct ConfigMetadata: Codable {
    let dpi: Double
    let languages: [String]
    let forceOCR: Bool
    
    enum CodingKeys: String, CodingKey {
        case dpi
        case languages
        case forceOCR = "force_ocr"
    }
}

struct OCRSummary: Codable {
    let totalChars: Int
    let pagesDirect: Int
    let pagesOCR: Int
    let avgConfidence: Double
    let warnings: [String]
    
    enum CodingKeys: String, CodingKey {
        case totalChars = "total_chars"
        case pagesDirect = "pages_direct"
        case pagesOCR = "pages_ocr"
        case avgConfidence = "avg_confidence"
        case warnings
    }
}

extension OCRResult {
    static func create(pages: [PageResult], config: OCRConfig) -> OCRResult {
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
        
        return OCRResult(
            metadata: metadata,
            pages: pages,
            summary: summary
        )
    }
}

