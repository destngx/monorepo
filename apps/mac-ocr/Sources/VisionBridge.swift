import Foundation
import Vision
import CoreGraphics

class VisionBridge {
    static func recognizeText(cgImage: CGImage, config: OCRConfig) throws -> (content: String, confidence: Double, bboxes: [BBox]?) {
        let request = VNRecognizeTextRequest()
        request.recognitionLevel = .accurate
        request.usesLanguageCorrection = true
        request.recognitionLanguages = config.languages
        
        // Use revision 3 for best accuracy (macOS 12+)
        if #available(macOS 12.0, *) {
            request.revision = VNRecognizeTextRequestRevision3
        }
        
        let handler = VNImageRequestHandler(cgImage: cgImage, options: [:])
        try handler.perform([request])
        
        guard let results = request.results else {
            return ("", 0.0, nil)
        }
        
        var totalConfidence: Double = 0
        var observationCount = 0
        var fullText = ""
        var bboxes: [BBox] = []
        
        for observation in results {
            guard let candidate = observation.topCandidates(1).first else { continue }
            
            let text = candidate.string
            fullText += text + "\n"
            totalConfidence += Double(candidate.confidence)
            observationCount += 1
            
            if config.includeBBox {
                let box = observation.boundingBox
                bboxes.append(BBox(
                    text: text,
                    x: Double(box.origin.x),
                    y: Double(box.origin.y),
                    w: Double(box.size.width),
                    h: Double(box.size.height)
                ))
            }
        }
        
        let avgConfidence = observationCount > 0 ? totalConfidence / Double(observationCount) : 0.0
        return (fullText.trimmingCharacters(in: .whitespacesAndNewlines), avgConfidence, config.includeBBox ? bboxes : nil)
    }
}
