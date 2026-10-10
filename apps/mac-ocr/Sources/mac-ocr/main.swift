import Foundation
import MacOCRCore

func fail(_ error: Error) -> Never {
    let ocrError = error as? OCRError ?? .recognitionFailed(error.localizedDescription)
    fputs("Error: \(ocrError.localizedDescription)\n", stderr)
    if case .usage = ocrError {
        fputs("\n\(CommandLineParser.usage)\n", stderr)
    }
    exit(ocrError.exitCode)
}

do {
    switch try CommandLineParser.parse(Array(CommandLine.arguments.dropFirst())) {
    case .help:
        print(CommandLineParser.usage)
    case .version:
        print("mac-ocr \(MacOCR.version)")
    case .run(let options):
        let urls = options.inputs.map { URL(fileURLWithPath: $0) }
        let results = try Extractor.extract(urls: urls, options: options.ocr)
        for result in results {
            for warning in result.summary.warnings {
                fputs("Warning: \(result.metadata.filename): \(warning)\n", stderr)
            }
        }
        print(try OutputFormatter.format(results, as: options.format))
    }
} catch {
    fail(error)
}
