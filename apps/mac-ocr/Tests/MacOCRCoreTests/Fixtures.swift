import CoreGraphics
import CoreText
import Foundation
import ImageIO
import PDFKit
import UniformTypeIdentifiers

/// Generates test documents on the fly so the suite has no binary fixtures.
enum Fixtures {
    static let sentence = "The quick brown fox jumps over the lazy dog"
    static let letter = CGSize(width: 612, height: 792)

    static func makeDirectory() throws -> URL {
        let url = FileManager.default.temporaryDirectory
            .appendingPathComponent("mac-ocr-tests-\(UUID().uuidString)", isDirectory: true)
        try FileManager.default.createDirectory(at: url, withIntermediateDirectories: true)
        return url
    }

    /// Draws a few lines of the sentence in `size` (points), scaled by `scale`.
    static func drawText(in context: CGContext, size: CGSize, scale: CGFloat = 1) {
        let font = CTFontCreateWithName("Helvetica" as CFString, 14 * scale, nil)
        let attributes = [kCTFontAttributeName: font] as CFDictionary
        var y = size.height * scale - 72 * scale
        for i in 1...6 {
            let string = CFAttributedStringCreate(nil, "\(i). \(sentence)" as CFString, attributes)!
            context.textPosition = CGPoint(x: 54 * scale, y: y)
            CTLineDraw(CTLineCreateWithAttributedString(string), context)
            y -= 28 * scale
        }
    }

    /// A PDF whose pages carry a real text layer.
    static func textPDF(at url: URL, pages: Int = 1) {
        var box = CGRect(origin: .zero, size: letter)
        let context = CGContext(url as CFURL, mediaBox: &box, nil)!
        for _ in 0..<pages {
            context.beginPDFPage(nil)
            drawText(in: context, size: letter)
            context.endPDFPage()
        }
        context.closePDF()
    }

    /// A rasterized page of text with no text layer, as a scanner would produce.
    static func scanImage(size: CGSize = letter, dpi: CGFloat = 200) -> CGImage {
        let scale = dpi / 72
        let width = Int(size.width * scale), height = Int(size.height * scale)
        let context = CGContext(
            data: nil, width: width, height: height, bitsPerComponent: 8, bytesPerRow: 0,
            space: CGColorSpaceCreateDeviceGray(), bitmapInfo: CGImageAlphaInfo.none.rawValue
        )!
        context.setFillColor(gray: 1, alpha: 1)
        context.fill(CGRect(x: 0, y: 0, width: width, height: height))
        drawText(in: context, size: size, scale: scale)
        return context.makeImage()!
    }

    static func scannedPDF(at url: URL, pages: Int = 1) {
        var box = CGRect(origin: .zero, size: letter)
        let image = scanImage()
        let context = CGContext(url as CFURL, mediaBox: &box, nil)!
        for _ in 0..<pages {
            context.beginPDFPage(nil)
            context.draw(image, in: box)
            context.endPDFPage()
        }
        context.closePDF()
    }

    /// A landscape media box whose text only reads upright once the page's /Rotate 90 is applied.
    static func rotatedTextPDF(at url: URL) {
        var box = CGRect(x: 0, y: 0, width: letter.height, height: letter.width)
        let context = CGContext(url as CFURL, mediaBox: &box, nil)!
        context.beginPDFPage(nil)
        context.translateBy(x: letter.height, y: 0)
        context.rotate(by: .pi / 2)
        drawText(in: context, size: letter)
        context.endPDFPage()
        context.closePDF()

        let document = PDFDocument(url: url)!
        document.page(at: 0)!.rotation = 90
        document.write(to: url)
    }

    /// A PDF whose crop box is the top half of the media box.
    static func croppedPDF(at url: URL) {
        textPDF(at: url)
        let document = PDFDocument(url: url)!
        let page = document.page(at: 0)!
        page.setBounds(CGRect(x: 0, y: letter.height / 2, width: letter.width, height: letter.height / 2), for: .cropBox)
        document.write(to: url)
    }

    /// Writes `image` to `url`, tagging it with an EXIF `orientation`.
    static func writeImage(_ image: CGImage, to url: URL, type: UTType, orientation: CGImagePropertyOrientation = .up) {
        let destination = CGImageDestinationCreateWithURL(url as CFURL, type.identifier as CFString, 1, nil)!
        let properties = [kCGImagePropertyOrientation: orientation.rawValue] as CFDictionary
        CGImageDestinationAddImage(destination, image, properties)
        CGImageDestinationFinalize(destination)
    }

    /// Rotates `image` 90 degrees clockwise, so it reads upright only when displayed with EXIF `.left`.
    static func rotatedClockwise(_ image: CGImage) -> CGImage {
        let context = CGContext(
            data: nil, width: image.height, height: image.width, bitsPerComponent: 8, bytesPerRow: 0,
            space: CGColorSpaceCreateDeviceGray(), bitmapInfo: CGImageAlphaInfo.none.rawValue
        )!
        context.translateBy(x: 0, y: CGFloat(image.width))
        context.rotate(by: -.pi / 2)
        context.draw(image, in: CGRect(x: 0, y: 0, width: image.width, height: image.height))
        return context.makeImage()!
    }
}
