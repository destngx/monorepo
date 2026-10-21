import CoreGraphics
import CoreImage
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

    /// A ruled two-column table (name, value) drawn at 200 dpi.
    static let tableRows = [("Glucose", "6.4"), ("Cholesterol", "5.82"), ("Creatinin", "88"), ("Triglycerid", "1.96")]

    static func tableImage() -> CGImage {
        let scale: CGFloat = 200 / 72
        let size = letter
        let context = CGContext(
            data: nil, width: Int(size.width * scale), height: Int(size.height * scale), bitsPerComponent: 8, bytesPerRow: 0,
            space: CGColorSpaceCreateDeviceGray(), bitmapInfo: CGImageAlphaInfo.none.rawValue
        )!
        context.scaleBy(x: scale, y: scale)
        context.setFillColor(gray: 1, alpha: 1)
        context.fill(CGRect(origin: .zero, size: size))
        let font = CTFontCreateWithName("Helvetica" as CFString, 14, nil)
        let attributes = [kCTFontAttributeName: font] as CFDictionary
        let top: CGFloat = 700, rowHeight: CGFloat = 30
        for (i, (name, value)) in tableRows.enumerated() {
            let y = top - CGFloat(i) * rowHeight
            for (text, x) in [(name, CGFloat(80)), (value, CGFloat(360))] {
                context.textPosition = CGPoint(x: x, y: y)
                CTLineDraw(CTLineCreateWithAttributedString(CFAttributedStringCreate(nil, text as CFString, attributes)!), context)
            }
            context.stroke(CGRect(x: 70, y: y - 9, width: 440, height: rowHeight))
        }
        context.stroke(CGRect(x: 70, y: top - 9 - CGFloat(tableRows.count - 1) * rowHeight, width: 280, height: CGFloat(tableRows.count) * rowHeight))
        return context.makeImage()!
    }

    /// The scan fixture with a QR code in its lower-right quarter; returns the code's normalized rect too.
    static func scanWithQRCode(payload: String) -> (image: CGImage, qr: CGRect) {
        let page = scanImage()
        let code = CIFilter(name: "CIQRCodeGenerator", parameters: ["inputMessage": Data(payload.utf8), "inputCorrectionLevel": "M"])!
            .outputImage!.transformed(by: CGAffineTransform(scaleX: 12, y: 12))
        let codeImage = CIContext().createCGImage(code, from: code.extent)!
        let context = CGContext(
            data: nil, width: page.width, height: page.height, bitsPerComponent: 8, bytesPerRow: 0,
            space: CGColorSpaceCreateDeviceGray(), bitmapInfo: CGImageAlphaInfo.none.rawValue
        )!
        context.draw(page, in: CGRect(x: 0, y: 0, width: page.width, height: page.height))
        let rect = CGRect(x: page.width - codeImage.width - 150, y: 150, width: codeImage.width, height: codeImage.height)
        context.draw(codeImage, in: rect)
        let normalized = CGRect(
            x: rect.minX / CGFloat(page.width), y: rect.minY / CGFloat(page.height),
            width: rect.width / CGFloat(page.width), height: rect.height / CGFloat(page.height)
        )
        return (context.makeImage()!, normalized)
    }

    /// Reads a written page image's pixel size and color model.
    static func imageInfo(at url: URL) -> (width: Int, height: Int, colorModel: String, profile: String, depth: Int, type: String)? {
        guard let source = CGImageSourceCreateWithURL(url as CFURL, nil),
              let properties = CGImageSourceCopyPropertiesAtIndex(source, 0, nil) as? [CFString: Any],
              let width = properties[kCGImagePropertyPixelWidth] as? Int,
              let height = properties[kCGImagePropertyPixelHeight] as? Int else { return nil }
        return (
            width, height,
            properties[kCGImagePropertyColorModel] as? String ?? "",
            properties[kCGImagePropertyProfileName] as? String ?? "",
            properties[kCGImagePropertyDepth] as? Int ?? 0,
            CGImageSourceGetType(source) as String? ?? ""
        )
    }

    /// Mean gray level (0 black, 255 white) of a normalized, bottom-left-origin rect of the image at `url`.
    static func meanGray(at url: URL, in rect: CGRect) -> Double {
        let source = CGImageSourceCreateWithURL(url as CFURL, nil)!
        let image = CGImageSourceCreateImageAtIndex(source, 0, nil)!
        let (w, h) = (image.width, image.height)
        let context = CGContext(
            data: nil, width: w, height: h, bitsPerComponent: 8, bytesPerRow: w,
            space: CGColorSpaceCreateDeviceGray(), bitmapInfo: CGImageAlphaInfo.none.rawValue
        )!
        context.draw(image, in: CGRect(x: 0, y: 0, width: w, height: h))
        let pixels = context.data!.bindMemory(to: UInt8.self, capacity: w * h)
        let x0 = Int(rect.minX * CGFloat(w)), x1 = Int(rect.maxX * CGFloat(w))
        // Bitmap rows run top to bottom; the rect has a bottom-left origin.
        let y0 = Int((1 - rect.maxY) * CGFloat(h)), y1 = Int((1 - rect.minY) * CGFloat(h))
        let rows = max(0, y0)..<max(max(0, y0), min(h, y1))
        let columns = max(0, x0)..<max(max(0, x0), min(w, x1))
        var total = 0, count = 0
        for y in rows {
            for x in columns {
                total += Int(pixels[y * w + x])
                count += 1
            }
        }
        return count == 0 ? 255 : Double(total) / Double(count)
    }
}
