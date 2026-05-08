import Foundation

func printUsage() {
    let usage = """
    USAGE: mac-ocr <input> [OPTIONS]

    ARGUMENTS:
      <input>           Path to image or PDF file

    OPTIONS:
      --format <fmt>    Output format: plain | json  [default: plain]
      --lang <langs>    Comma-separated language codes  [default: en-US]
      --dpi <n>         Render DPI for PDF pages (72–600)  [default: 150]
      --force-ocr       Skip direct text extraction; always use Vision OCR
      --bbox            Include bounding boxes in JSON output
      --timeout <secs>  Per-page timeout in seconds  [default: 30]
      --page <n>        Extract only page N (1-indexed, PDFs only)
      --version         Print version string
      -h, --help        Show help
    """
    fputs(usage + "\n", stderr)
}

func main() {
    let args = CommandLine.arguments
    
    if args.contains("-h") || args.contains("--help") {
        printUsage()
        exit(0)
    }
    
    if args.contains("--version") {
        print("mac-ocr 1.0.0")
        exit(0)
    }
    
    guard args.count > 1 else {
        printUsage()
        exit(1)
    }
    
    let inputPathStr = args[1]
    if inputPathStr.hasPrefix("-") {
        fputs("Error: Missing input path\n", stderr)
        printUsage()
        exit(1)
    }
    
    let inputPath = URL(fileURLWithPath: inputPathStr)
    
    // Default config
    var format: OutputFormat = .plain
    var languages: [String] = ["en-US"]
    var dpi: CGFloat = 150
    var forceOCR = false
    var includeBBox = false
    var timeout: TimeInterval = 30
    var targetPage: Int? = nil
    
    // Parse options
    var i = 2
    while i < args.count {
        let arg = args[i]
        switch arg {
        case "--format":
            if i + 1 < args.count {
                format = OutputFormat(rawValue: args[i+1]) ?? .plain
                i += 1
            }
        case "--lang":
            if i + 1 < args.count {
                languages = args[i+1].split(separator: ",").map { String($0).trimmingCharacters(in: .whitespaces) }
                i += 1
            }
        case "--dpi":
            if i + 1 < args.count, let val = Double(args[i+1]) {
                dpi = CGFloat(val)
                i += 1
            }
        case "--force-ocr":
            forceOCR = true
        case "--bbox":
            includeBBox = true
        case "--timeout":
            if i + 1 < args.count, let val = Double(args[i+1]) {
                timeout = val
                i += 1
            }
        case "--page":
            if i + 1 < args.count, let val = Int(args[i+1]) {
                targetPage = val
                i += 1
            }
        default:
            fputs("Warning: Unknown option \(arg)\n", stderr)
        }
        i += 1
    }
    
    let config = OCRConfig(
        inputPath: inputPath,
        format: format,
        languages: languages,
        dpi: dpi,
        forceOCR: forceOCR,
        includeBBox: includeBBox,
        timeout: timeout,
        targetPage: targetPage
    )
    
    do {
        let pages = try InputRouter.route(config: config)
        let output = Formatter.format(pages: pages, config: config)
        print(output)
        exit(0)
    } catch {
        let nsError = error as NSError
        fputs("Error: \(nsError.localizedDescription)\n", stderr)
        exit(Int32(nsError.code > 0 ? nsError.code : 4))
    }
}

main()
