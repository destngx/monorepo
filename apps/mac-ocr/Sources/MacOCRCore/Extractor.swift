import Foundation
import ImageIO
import UniformTypeIdentifiers
import Vision

public enum Extractor {
    /// Extracts files in input order. Inputs and languages are validated up front so bad arguments fail fast.
    /// Files run sequentially: Vision serializes recognition, so cross-file parallelism gains nothing.
    public static func extract(urls: [URL], options: OCROptions) throws -> [OCRResult] {
        try TextRecognizer.validate(languages: options.languages)
        for url in urls {
            try ensureFileExists(url)
        }
        return try urls.map { try extract(url: $0, options: options) }
    }

    static func extract(url: URL, options: OCROptions) throws -> OCRResult {
        let pages: [PageResult]
        switch try contentType(of: url) {
        case let type where type.conforms(to: .pdf):
            pages = try PDFPipeline.process(url: url, options: options)
        case let type where type.conforms(to: .image):
            pages = [try processImage(url: url, options: options)]
        case let type:
            throw OCRError.unsupportedType("Unsupported file type \(type.identifier): \(url.lastPathComponent)")
        }
        return OCRResult.make(url: url, pages: pages, options: options)
    }

    private static func ensureFileExists(_ url: URL) throws {
        var isDirectory: ObjCBool = false
        guard FileManager.default.fileExists(atPath: url.path, isDirectory: &isDirectory), !isDirectory.boolValue else {
            throw OCRError.fileNotFound("File not found: \(url.path)")
        }
    }

    private static func contentType(of url: URL) throws -> UTType {
        guard let type = try? url.resourceValues(forKeys: [.contentTypeKey]).contentType else {
            throw OCRError.unsupportedType("Could not determine file type: \(url.lastPathComponent)")
        }
        return type
    }

    private static func processImage(url: URL, options: OCROptions) throws -> PageResult {
        guard let source = CGImageSourceCreateWithURL(url as CFURL, nil),
              let image = CGImageSourceCreateImageAtIndex(source, 0, nil) else {
            throw OCRError.renderFailed("Failed to decode image: \(url.lastPathComponent)")
        }
        let properties = CGImageSourceCopyPropertiesAtIndex(source, 0, nil) as? [CFString: Any]
        let orientation = (properties?[kCGImagePropertyOrientation] as? UInt32)
            .flatMap(CGImagePropertyOrientation.init(rawValue:)) ?? .up

        let handler = VNImageRequestHandler(cgImage: image, orientation: orientation)
        let ocr = try TextRecognizer.recognize(handler, options: options)
        return PageResult(index: 1, method: .ocr, content: ocr.content, confidence: ocr.confidence, bbox: ocr.lines)
    }
}
