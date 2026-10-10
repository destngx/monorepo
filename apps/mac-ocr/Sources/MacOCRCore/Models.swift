import Foundation
import CoreGraphics

public enum MacOCR {
    public static let version = "1.1.0"
    public static let schemaVersion = "1.1"
}

public enum ExtractionMethod: String, Codable {
    case direct
    case ocr
}

public enum OutputFormat: String, CaseIterable {
    case plain
    case json
}

/// A recognized line. Coordinates are normalized (0-1) with a bottom-left origin (Vision convention).
public struct BBox: Codable, Equatable {
    public let text: String
    public let confidence: Double
    public let x: Double
    public let y: Double
    public let w: Double
    public let h: Double
}

public struct PageResult: Codable, Equatable {
    public let index: Int
    public let method: ExtractionMethod
    public let content: String
    public let confidence: Double
    public let charCount: Int
    public let bbox: [BBox]?
    /// Set when the page could not be extracted; `content` is empty in that case.
    public let error: String?

    enum CodingKeys: String, CodingKey {
        case index
        case method
        case content
        case confidence
        case charCount = "char_count"
        case bbox
        case error
    }

    init(index: Int, method: ExtractionMethod, content: String, confidence: Double, bbox: [BBox]?, error: String? = nil) {
        self.index = index
        self.method = method
        self.content = content
        self.confidence = confidence
        self.charCount = content.count
        self.bbox = bbox
        self.error = error
    }

    static func failed(index: Int, error: OCRError) -> PageResult {
        PageResult(index: index, method: .ocr, content: "", confidence: 0, bbox: nil, error: error.localizedDescription)
    }
}

public struct OCROptions: Equatable {
    public var languages: [String] = ["en-US"]
    public var dpi: CGFloat = 150
    public var forceOCR = false
    public var includeBBox = false
    public var timeout: TimeInterval = 30
    /// 1-indexed page to extract (PDFs only).
    public var targetPage: Int?

    public init() {}
}

public struct OCRResult: Codable {
    public let schemaVersion: String
    public let metadata: OCRMetadata
    public let pages: [PageResult]
    public let summary: OCRSummary

    enum CodingKeys: String, CodingKey {
        case schemaVersion = "schema_version"
        case metadata
        case pages
        case summary
    }
}

public struct OCRMetadata: Codable {
    public let filename: String
    public let fileSize: Int64
    public let pageCount: Int
    public let processedAt: String
    public let toolVersion: String
    public let config: ConfigMetadata

    enum CodingKeys: String, CodingKey {
        case filename
        case fileSize = "file_size_bytes"
        case pageCount = "page_count"
        case processedAt = "processed_at"
        case toolVersion = "tool_version"
        case config
    }
}

public struct ConfigMetadata: Codable {
    public let dpi: Double
    public let languages: [String]
    public let forceOCR: Bool

    enum CodingKeys: String, CodingKey {
        case dpi
        case languages
        case forceOCR = "force_ocr"
    }
}

public struct OCRSummary: Codable {
    public let totalChars: Int
    public let pagesDirect: Int
    public let pagesOCR: Int
    public let avgConfidence: Double
    public let warnings: [String]

    enum CodingKeys: String, CodingKey {
        case totalChars = "total_chars"
        case pagesDirect = "pages_direct"
        case pagesOCR = "pages_ocr"
        case avgConfidence = "avg_confidence"
        case warnings
    }
}

extension OCRResult {
    static func make(url: URL, pages: [PageResult], options: OCROptions, processedAt: Date = Date()) -> OCRResult {
        let warnings = pages.compactMap { page in page.error.map { "page \(page.index): \($0)" } }
        let summary = OCRSummary(
            totalChars: pages.reduce(0) { $0 + $1.charCount },
            pagesDirect: pages.filter { $0.method == .direct }.count,
            pagesOCR: pages.filter { $0.method == .ocr }.count,
            avgConfidence: pages.isEmpty ? 0 : pages.reduce(0.0) { $0 + $1.confidence } / Double(pages.count),
            warnings: warnings
        )

        let metadata = OCRMetadata(
            filename: url.lastPathComponent,
            fileSize: (try? url.resourceValues(forKeys: [.fileSizeKey]).fileSize).map { Int64($0) } ?? 0,
            pageCount: pages.count,
            processedAt: ISO8601DateFormatter().string(from: processedAt),
            toolVersion: MacOCR.version,
            config: ConfigMetadata(
                dpi: Double(options.dpi),
                languages: options.languages,
                forceOCR: options.forceOCR
            )
        )

        return OCRResult(schemaVersion: MacOCR.schemaVersion, metadata: metadata, pages: pages, summary: summary)
    }
}
