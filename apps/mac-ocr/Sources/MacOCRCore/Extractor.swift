import Foundation
import ImageIO
import UniformTypeIdentifiers
import Vision

public enum Extractor {
    /// Extracts files in input order. Inputs and languages are validated up front so bad arguments fail fast.
    /// Files run sequentially: Vision serializes recognition, so cross-file parallelism gains nothing.
    public static func extract(urls: [URL], options: OCROptions) throws -> [OCRResult] {
        try TextRecognizer.validate(languages: options.languages)
        if let directory = options.renderDirectory {
            // Page images are named page-<n>.jpg, so a second input would overwrite the first one's.
            guard urls.count == 1 else { throw OCRError.usage("--render-dir takes exactly one input file") }
            try PageImageWriter.prepare(directory: directory)
        }
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

        let pageImage = try options.renderDirectory == nil ? nil : PageImageWriter.write(
            try upright(source, image: image, url: url), page: 1, options: options
        )

        let ocr = try TextRecognizer.recognize(image, orientation: orientation, options: options)
        return PageResult(
            index: 1,
            method: .ocr,
            content: ocr.content,
            confidence: ocr.confidence,
            bbox: ocr.lines,
            barcodes: ocr.barcodes,
            skewDegrees: ocr.skewDegrees,
            image: pageImage,
            warnings: ocr.warnings
        )
    }

    /// The full-size image with its EXIF orientation applied, matching the space Vision reports coordinates in.
    private static func upright(_ source: CGImageSource, image: CGImage, url: URL) throws -> CGImage {
        let options = [
            kCGImageSourceCreateThumbnailFromImageAlways: true,
            kCGImageSourceCreateThumbnailWithTransform: true,
            kCGImageSourceThumbnailMaxPixelSize: max(image.width, image.height),
        ] as CFDictionary
        guard let upright = CGImageSourceCreateThumbnailAtIndex(source, 0, options) else {
            throw OCRError.renderFailed("Failed to orient image: \(url.lastPathComponent)")
        }
        return upright
    }
}

extension CGImagePropertyOrientation {
    /// Orientations that turn the stored image a quarter turn, so width and height trade places.
    var swapsAxes: Bool {
        switch self {
        case .left, .leftMirrored, .right, .rightMirrored: return true
        default: return false
        }
    }
}
