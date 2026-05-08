import Foundation
import UniformTypeIdentifiers
import AppKit

class InputRouter {
    static func route(config: OCRConfig) throws -> [PageResult] {
        let url = config.inputPath
        
        guard let values = try? url.resourceValues(forKeys: [.contentTypeKey]),
              let utType = values.contentType else {
            throw NSError(domain: "mac-ocr", code: 3, userInfo: [NSLocalizedDescriptionKey: "Could not determine file type for \(url.lastPathComponent)"])
        }
        
        if utType.conforms(to: .pdf) {
            return try PDFPipeline.process(url: url, config: config)
        } else if utType.conforms(to: .image) {
            return try processImage(url: url, config: config)
        } else {
            throw NSError(domain: "mac-ocr", code: 3, userInfo: [NSLocalizedDescriptionKey: "Unsupported file type: \(utType.identifier)"])
        }
    }
    
    private static func processImage(url: URL, config: OCRConfig) throws -> [PageResult] {
        guard let image = NSImage(contentsOf: url),
              let cgImage = image.cgImage(forProposedRect: nil, context: nil, hints: nil) else {
            throw NSError(domain: "mac-ocr", code: 2, userInfo: [NSLocalizedDescriptionKey: "Failed to load image at \(url.path)"])
        }
        
        let ocr = try VisionBridge.recognizeText(cgImage: cgImage, config: config)
        let result = PageResult(
            index: 1,
            method: .ocr,
            content: ocr.content,
            confidence: ocr.confidence,
            charCount: ocr.content.count,
            bbox: ocr.bboxes
        )
        return [result]
    }
}
