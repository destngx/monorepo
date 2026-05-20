import Foundation

func printUsage() {
    let usage = """
    USAGE: mac-ocr <input-files...> [OPTIONS]

    ARGUMENTS:
      <input-files...>  One or more paths to image or PDF files

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
    
    var inputPathsStr: [String] = []
    var format: OutputFormat = .plain
    var languages: [String] = ["en-US"]
    var dpi: CGFloat = 150
    var forceOCR = false
    var includeBBox = false
    var timeout: TimeInterval = 30
    var targetPage: Int? = nil
    
    var i = 1
    while i < args.count {
        let arg = args[i]
        switch arg {
        case "--format":
            if i + 1 < args.count {
                format = OutputFormat(rawValue: args[i+1]) ?? .plain
                i += 2
            } else {
                fputs("Error: Missing value for --format\n", stderr)
                exit(1)
            }
        case "--lang":
            if i + 1 < args.count {
                languages = args[i+1].split(separator: ",").map { String($0).trimmingCharacters(in: .whitespaces) }
                i += 2
            } else {
                fputs("Error: Missing value for --lang\n", stderr)
                exit(1)
            }
        case "--dpi":
            if i + 1 < args.count {
                if let val = Double(args[i+1]) {
                    dpi = CGFloat(val)
                } else {
                    fputs("Error: Invalid value for --dpi\n", stderr)
                    exit(1)
                }
                i += 2
            } else {
                fputs("Error: Missing value for --dpi\n", stderr)
                exit(1)
            }
        case "--force-ocr":
            forceOCR = true
            i += 1
        case "--bbox":
            includeBBox = true
            i += 1
        case "--timeout":
            if i + 1 < args.count {
                if let val = Double(args[i+1]) {
                    timeout = val
                } else {
                    fputs("Error: Invalid value for --timeout\n", stderr)
                    exit(1)
                }
                i += 2
            } else {
                fputs("Error: Missing value for --timeout\n", stderr)
                exit(1)
            }
        case "--page":
            if i + 1 < args.count {
                if let val = Int(args[i+1]) {
                    targetPage = val
                } else {
                    fputs("Error: Invalid value for --page\n", stderr)
                    exit(1)
                }
                i += 2
            } else {
                fputs("Error: Missing value for --page\n", stderr)
                exit(1)
            }
        default:
            if arg.hasPrefix("-") {
                fputs("Error: Unknown option \(arg)\n", stderr)
                printUsage()
                exit(1)
            } else {
                inputPathsStr.append(arg)
                i += 1
            }
        }
    }
    
    guard !inputPathsStr.isEmpty else {
        fputs("Error: Missing input path\n", stderr)
        printUsage()
        exit(1)
    }
    
    // Validate file existence and make sure they are not directories
    var inputPaths: [URL] = []
    for pathStr in inputPathsStr {
        let url = URL(fileURLWithPath: pathStr)
        var isDirectory: ObjCBool = false
        if !FileManager.default.fileExists(atPath: url.path, isDirectory: &isDirectory) || isDirectory.boolValue {
            fputs("Error: File not found: \(pathStr)\n", stderr)
            exit(2)
        }
        inputPaths.append(url)
    }
    
    do {
        var results: [OCRResult] = []
        for inputPath in inputPaths {
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
            
            let pages = try InputRouter.route(config: config)
            let result = OCRResult.create(pages: pages, config: config)
            results.append(result)
        }
        
        let output = Formatter.format(results: results, format: format)
        print(output)
        exit(0)
    } catch {
        let nsError = error as NSError
        fputs("Error: \(nsError.localizedDescription)\n", stderr)
        exit(Int32(nsError.code > 0 ? nsError.code : 4))
    }
}

main()
