import Foundation

/// Errors surfaced by mac-ocr. Each case maps to a documented process exit code.
public enum OCRError: Error, Equatable, LocalizedError {
    case usage(String)
    case fileNotFound(String)
    case unsupportedType(String)
    case recognitionFailed(String)
    case renderFailed(String)
    case timeout(String)

    public var exitCode: Int32 {
        switch self {
        case .usage: return 1
        case .fileNotFound: return 2
        case .unsupportedType: return 3
        case .recognitionFailed: return 4
        case .renderFailed: return 5
        case .timeout: return 6
        }
    }

    /// Same error kind (and exit code) with `context` prepended to the message.
    func prefixed(_ context: String) -> OCRError {
        let message = "\(context): \(errorDescription ?? "")"
        switch self {
        case .usage: return .usage(message)
        case .fileNotFound: return .fileNotFound(message)
        case .unsupportedType: return .unsupportedType(message)
        case .recognitionFailed: return .recognitionFailed(message)
        case .renderFailed: return .renderFailed(message)
        case .timeout: return .timeout(message)
        }
    }

    public var errorDescription: String? {
        switch self {
        case .usage(let message),
             .fileNotFound(let message),
             .unsupportedType(let message),
             .recognitionFailed(let message),
             .renderFailed(let message),
             .timeout(let message):
            return message
        }
    }
}
