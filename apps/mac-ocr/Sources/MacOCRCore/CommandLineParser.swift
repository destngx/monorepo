import Foundation

public enum Command: Equatable {
    case help
    case version
    case run(RunOptions)
}

public struct RunOptions: Equatable {
    public var inputs: [String] = []
    public var format: OutputFormat = .plain
    public var ocr = OCROptions()
}

public enum CommandLineParser {
    public static let dpiRange: ClosedRange<CGFloat> = 72...600

    public static let usage = """
    USAGE: mac-ocr <input-files...> [OPTIONS]

    ARGUMENTS:
      <input-files...>  One or more paths to image or PDF files

    OPTIONS:
      --format <fmt>    Output format: plain | json  [default: plain]
      --lang <langs>    Comma-separated Vision language codes  [default: en-US]
      --dpi <n>         Render DPI for PDF pages (72-600)  [default: 150]
      --force-ocr       Skip direct text extraction; always use Vision OCR
      --bbox            Include line bounding boxes in JSON output
      --timeout <secs>  Per-page recognition timeout in seconds  [default: 30]
      --page <n>        Extract only page N (1-indexed, PDFs only)
      --version         Print version string
      -h, --help        Show help
    """

    /// Parses arguments (excluding the executable name). Arguments after `--` are always treated as inputs.
    public static func parse(_ arguments: [String]) throws -> Command {
        var options = RunOptions()
        var iterator = arguments.makeIterator()

        func value(for flag: String) throws -> String {
            guard let next = iterator.next() else { throw OCRError.usage("Missing value for \(flag)") }
            return next
        }

        while let arg = iterator.next() {
            switch arg {
            case "-h", "--help":
                return .help
            case "--version":
                return .version
            case "--format":
                let raw = try value(for: arg)
                guard let format = OutputFormat(rawValue: raw) else {
                    throw OCRError.usage("Invalid value for --format: \(raw) (expected plain | json)")
                }
                options.format = format
            case "--lang":
                let languages = try value(for: arg)
                    .split(separator: ",")
                    .map { $0.trimmingCharacters(in: .whitespaces) }
                    .filter { !$0.isEmpty }
                guard !languages.isEmpty else { throw OCRError.usage("--lang requires at least one language code") }
                options.ocr.languages = languages
            case "--dpi":
                let raw = try value(for: arg)
                guard let dpi = Double(raw).map({ CGFloat($0) }), dpiRange.contains(dpi) else {
                    throw OCRError.usage("Invalid value for --dpi: \(raw) (expected 72-600)")
                }
                options.ocr.dpi = dpi
            case "--timeout":
                let raw = try value(for: arg)
                guard let timeout = TimeInterval(raw), timeout > 0, timeout.isFinite else {
                    throw OCRError.usage("Invalid value for --timeout: \(raw) (expected a positive number of seconds)")
                }
                options.ocr.timeout = timeout
            case "--page":
                let raw = try value(for: arg)
                guard let page = Int(raw), page >= 1 else {
                    throw OCRError.usage("Invalid value for --page: \(raw) (expected an integer >= 1)")
                }
                options.ocr.targetPage = page
            case "--force-ocr":
                options.ocr.forceOCR = true
            case "--bbox":
                options.ocr.includeBBox = true
            case "--":
                while let input = iterator.next() { options.inputs.append(input) }
            default:
                guard !arg.hasPrefix("-") else { throw OCRError.usage("Unknown option \(arg)") }
                options.inputs.append(arg)
            }
        }

        guard !options.inputs.isEmpty else { throw OCRError.usage("Missing input path") }
        return .run(options)
    }
}
