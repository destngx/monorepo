import Foundation
import PDFKit
import Vision
import AppKit

class PDFPipeline {
    static func process(url: URL, config: OCRConfig) throws -> [PageResult] {
        guard let document = PDFDocument(url: url) else {
            throw NSError(domain: "mac-ocr", code: 5, userInfo: [NSLocalizedDescriptionKey: "Failed to load PDF document"])
        }
        
        let totalPages = document.pageCount
        var pageIndices: [Int] = []
        
        if let target = config.targetPage {
            if target >= 1 && target <= totalPages {
                pageIndices = [target - 1]
            } else {
                throw NSError(domain: "mac-ocr", code: 1, userInfo: [NSLocalizedDescriptionKey: "Target page \(target) is out of range (1-\(totalPages))"])
            }
        } else {
            pageIndices = Array(0..<totalPages)
        }
        
        var results = [PageResult?](repeating: nil, count: pageIndices.count)
        let resultsLock = NSLock()
        
        // Use parallel processing for documents with more than 4 pages
        let isParallel = pageIndices.count > 4
        
        if isParallel {
            DispatchQueue.concurrentPerform(iterations: pageIndices.count) { i in
                let pageIndex = pageIndices[i]
                if let page = document.page(at: pageIndex) {
                    let result = processPage(page, index: pageIndex + 1, config: config)
                    resultsLock.lock()
                    results[i] = result
                    resultsLock.unlock()
                }
            }
        } else {
            for (i, pageIndex) in pageIndices.enumerated() {
                if let page = document.page(at: pageIndex) {
                    results[i] = processPage(page, index: pageIndex + 1, config: config)
                }
            }
        }
        
        return results.compactMap { $0 }
    }
    
    private static func processPage(_ page: PDFPage, index: Int, config: OCRConfig) -> PageResult {
        if !config.forceOCR {
            if let directText = page.string {
                let printableChars = directText.compactMap { $0.isWhitespace || $0.isPunctuation || $0.isLetter || $0.isNumber ? $0 : nil }.count
                let alnumChars = directText.filter { $0.isLetter || $0.isNumber }.count
                let density = printableChars > 0 ? Double(alnumChars) / Double(printableChars) : 0.0
                
                if printableChars >= 50 && density >= 0.40 {
                    return PageResult(
                        index: index,
                        method: .direct,
                        content: directText.trimmingCharacters(in: .whitespacesAndNewlines),
                        confidence: 1.0,
                        charCount: directText.count,
                        bbox: nil
                    )
                }
            }
        }
        
        // Fallback to OCR
        guard let cgImage = renderPage(page, dpi: config.dpi) else {
            return PageResult(index: index, method: .ocr, content: "[Error: Rendering failed]", confidence: 0.0, charCount: 0, bbox: nil)
        }
        
        do {
            let ocr = try VisionBridge.recognizeText(cgImage: cgImage, config: config)
            return PageResult(
                index: index,
                method: .ocr,
                content: ocr.content,
                confidence: ocr.confidence,
                charCount: ocr.content.count,
                bbox: ocr.bboxes
            )
        } catch {
            return PageResult(index: index, method: .ocr, content: "[Error: \(error.localizedDescription)]", confidence: 0.0, charCount: 0, bbox: nil)
        }
    }
    
    private static func renderPage(_ page: PDFPage, dpi: CGFloat) -> CGImage? {
        let pageRect = page.bounds(for: .mediaBox)
        let scale = dpi / 72.0
        let size = CGSize(width: pageRect.width * scale, height: pageRect.height * scale)
        
        let image = NSImage(size: size)
        image.lockFocus()
        
        guard let context = NSGraphicsContext.current?.cgContext else {
            image.unlockFocus()
            return nil
        }
        
        context.setFillColor(NSColor.white.cgColor)
        context.fill(CGRect(origin: .zero, size: size))
        
        context.scaleBy(x: scale, y: scale)
        context.translateBy(x: -pageRect.origin.x, y: -pageRect.origin.y)
        
        page.draw(with: .mediaBox, to: context)
        
        image.unlockFocus()
        
        return image.cgImage(forProposedRect: nil, context: nil, hints: nil)
    }
}
