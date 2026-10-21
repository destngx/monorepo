import CoreGraphics
import Foundation
import ImageIO
import UniformTypeIdentifiers

/// Writes page images for `--render-dir`: upright 8-bit sRGB JPEGs named `page-<n>.jpg`.
enum PageImageWriter {
    static let qualityRange: ClosedRange<Double> = 0.1...1.0

    static func url(page: Int, in directory: URL) -> URL {
        directory.appendingPathComponent("page-\(page).jpg")
    }

    static func prepare(directory: URL) throws {
        do {
            try FileManager.default.createDirectory(at: directory, withIntermediateDirectories: true)
        } catch {
            throw OCRError.renderFailed("Failed to create render directory \(directory.path): \(error.localizedDescription)")
        }
    }

    static func write(_ image: CGImage, page: Int, options: OCROptions) throws -> PageImage? {
        guard let directory = options.renderDirectory else { return nil }
        let url = url(page: page, in: directory)
        guard let srgb = redraw(image, space: CGColorSpace(name: CGColorSpace.sRGB)!, bitmapInfo: CGImageAlphaInfo.noneSkipLast.rawValue),
              let destination = CGImageDestinationCreateWithURL(url as CFURL, UTType.jpeg.identifier as CFString, 1, nil) else {
            throw OCRError.renderFailed("Failed to create page image \(url.path)")
        }
        let properties = [kCGImageDestinationLossyCompressionQuality: options.jpegQuality] as CFDictionary
        CGImageDestinationAddImage(destination, srgb, properties)
        guard CGImageDestinationFinalize(destination) else {
            throw OCRError.renderFailed("Failed to write page image \(url.path)")
        }
        return PageImage(path: url.path, width: srgb.width, height: srgb.height)
    }

    /// An 8-bit grayscale copy; OCR runs on grayscale whether or not a color page is written.
    static func grayscale(_ image: CGImage) -> CGImage? {
        redraw(image, space: CGColorSpaceCreateDeviceGray(), bitmapInfo: CGImageAlphaInfo.none.rawValue)
    }

    private static func redraw(_ image: CGImage, space: CGColorSpace, bitmapInfo: UInt32) -> CGImage? {
        guard let context = CGContext(
            data: nil,
            width: image.width,
            height: image.height,
            bitsPerComponent: 8,
            bytesPerRow: 0,
            space: space,
            bitmapInfo: bitmapInfo
        ) else { return nil }
        let rect = CGRect(x: 0, y: 0, width: image.width, height: image.height)
        // White under any transparency (PNG, HEIC with alpha), as on paper.
        context.setFillColor(CGColor(gray: 1, alpha: 1))
        context.fill(rect)
        context.draw(image, in: rect)
        return context.makeImage()
    }
}
