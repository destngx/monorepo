import Foundation
import CoreGraphics

public enum MacOCR {
    public static let version = "1.3.0"
    public static let schemaVersion = "1.3"
}

public enum ExtractionMethod: String, Codable {
    case direct
    case ocr
}

public enum OutputFormat: String, CaseIterable {
    case plain
    case json
}

/// A normalized (0-1) point with a bottom-left origin (Vision convention).
public struct Point: Codable, Equatable {
    public let x: Double
    public let y: Double
}

/// The four corners of a region as the recognizer reported them. On a tilted photo they follow the text,
/// unlike the axis-aligned `x`/`y`/`w`/`h` box, which then also covers parts of neighbouring lines.
/// Encoded as `[[x, y], [x, y], [x, y], [x, y]]`: top-left, top-right, bottom-right, bottom-left.
public struct Quad: Codable, Equatable {
    public let topLeft: Point
    public let topRight: Point
    public let bottomRight: Point
    public let bottomLeft: Point

    public func encode(to encoder: Encoder) throws {
        var container = encoder.singleValueContainer()
        try container.encode([topLeft, topRight, bottomRight, bottomLeft].map { [$0.x, $0.y] })
    }

    public init(from decoder: Decoder) throws {
        let corners = try decoder.singleValueContainer().decode([[Double]].self)
        guard corners.count == 4, corners.allSatisfy({ $0.count == 2 }) else {
            throw DecodingError.dataCorrupted(.init(codingPath: decoder.codingPath, debugDescription: "quad needs four [x, y] corners"))
        }
        let points = corners.map { Point(x: $0[0], y: $0[1]) }
        self.init(topLeft: points[0], topRight: points[1], bottomRight: points[2], bottomLeft: points[3])
    }

    /// Corners of an axis-aligned rectangle.
    init(_ rect: CGRect) {
        topLeft = Point(x: Double(rect.minX), y: Double(rect.maxY))
        topRight = Point(x: Double(rect.maxX), y: Double(rect.maxY))
        bottomRight = Point(x: Double(rect.maxX), y: Double(rect.minY))
        bottomLeft = Point(x: Double(rect.minX), y: Double(rect.minY))
    }

    init(topLeft: Point, topRight: Point, bottomRight: Point, bottomLeft: Point) {
        self.topLeft = topLeft
        self.topRight = topRight
        self.bottomRight = bottomRight
        self.bottomLeft = bottomLeft
    }
}

/// A word of a line (since 1.3): one whitespace-separated token with its own corners, so part of a line can be
/// masked without the rest.
public struct Word: Codable, Equatable {
    public let text: String
    public let quad: Quad
}

/// A recognized line. Coordinates are normalized (0-1) with a bottom-left origin (Vision convention).
public struct BBox: Codable, Equatable {
    public let text: String
    public let confidence: Double
    public let x: Double
    public let y: Double
    public let w: Double
    public let h: Double
    /// Since 1.2.
    public let quad: Quad?
    /// The line's whitespace-separated tokens in order (since 1.3). Nil when they could not all be boxed.
    public let words: [Word]?

    init(text: String, confidence: Double, rect: CGRect, quad: Quad, words: [Word]? = nil) {
        self.text = text
        self.confidence = confidence
        x = Double(rect.origin.x)
        y = Double(rect.origin.y)
        w = Double(rect.width)
        h = Double(rect.height)
        self.quad = quad
        self.words = words
    }
}

/// A detected barcode or QR code (since 1.2), reported so it can be masked. The payload is never output:
/// it can hold patient IDs.
public struct Barcode: Codable, Equatable {
    /// Vision symbology name, e.g. "QR", "Code128", "PDF417".
    public let symbology: String
    public let quad: Quad
}

/// An upright sRGB JPEG written with `--render-dir` (since 1.2). Line and barcode coordinates map onto it directly.
public struct PageImage: Codable, Equatable {
    public let path: String
    public let width: Int
    public let height: Int
}

public struct PageResult: Codable, Equatable {
    public let index: Int
    public let method: ExtractionMethod
    public let content: String
    public let confidence: Double
    public let charCount: Int
    public let bbox: [BBox]?
    /// Detected barcodes, with `--bbox` (since 1.2). Present, possibly empty, whenever detection ran.
    public let barcodes: [Barcode]?
    /// Estimated text skew in degrees (positive: text rises left to right) for OCR pages, with `--bbox` (since 1.2).
    public let skewDegrees: Double?
    /// The rendered page, with `--render-dir` (since 1.2).
    public let image: PageImage?
    /// Optional steps that did not finish, e.g. barcode detection (since 1.2). The page itself is valid.
    public let warnings: [String]?
    /// Set when the page could not be extracted; `content` is empty in that case.
    public let error: String?

    enum CodingKeys: String, CodingKey {
        case index
        case method
        case content
        case confidence
        case charCount = "char_count"
        case bbox
        case barcodes
        case skewDegrees = "skew_degrees"
        case image
        case warnings
        case error
    }

    init(
        index: Int,
        method: ExtractionMethod,
        content: String,
        confidence: Double,
        bbox: [BBox]?,
        barcodes: [Barcode]? = nil,
        skewDegrees: Double? = nil,
        image: PageImage? = nil,
        warnings: [String] = [],
        error: String? = nil
    ) {
        self.index = index
        self.method = method
        self.content = content
        self.confidence = confidence
        self.charCount = content.count
        self.bbox = bbox
        self.barcodes = barcodes
        self.skewDegrees = skewDegrees
        self.image = image
        self.warnings = warnings.isEmpty ? nil : warnings
        self.error = error
    }

    /// Keeps `image` when the page JPEG was written before the failure.
    static func failed(index: Int, error: OCRError, image: PageImage? = nil) -> PageResult {
        PageResult(index: index, method: .ocr, content: "", confidence: 0, bbox: nil, image: image, error: error.localizedDescription)
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
    public var layout: TextLayout = .rows
    /// Where to write `page-<n>.jpg` for every page; nil writes none.
    public var renderDirectory: URL?
    /// JPEG quality for rendered pages.
    public var jpegQuality: Double = 0.85

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
    /// Since 1.2.
    public let layout: TextLayout?

    enum CodingKeys: String, CodingKey {
        case dpi
        case languages
        case forceOCR = "force_ocr"
        case layout
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
        let warnings = pages.flatMap { page in
            ((page.warnings ?? []) + [page.error].compactMap { $0 }).map { "page \(page.index): \($0)" }
        }
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
                forceOCR: options.forceOCR,
                layout: options.layout
            )
        )

        return OCRResult(schemaVersion: MacOCR.schemaVersion, metadata: metadata, pages: pages, summary: summary)
    }
}
