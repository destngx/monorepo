import XCTest
@testable import MacOCRCore

final class CommandLineParserTests: XCTestCase {
    func testDefaults() throws {
        let options = try runOptions(["a.pdf"])
        XCTAssertEqual(options.inputs, ["a.pdf"])
        XCTAssertEqual(options.format, .plain)
        XCTAssertEqual(options.ocr, OCROptions())
    }

    func testAllOptions() throws {
        let options = try runOptions([
            "a.pdf", "--format", "json", "--lang", "en-US, vi-VT", "--dpi", "300",
            "--force-ocr", "--bbox", "--timeout", "2.5", "--page", "3", "b.png",
        ])
        XCTAssertEqual(options.inputs, ["a.pdf", "b.png"])
        XCTAssertEqual(options.format, .json)
        XCTAssertEqual(options.ocr.languages, ["en-US", "vi-VT"])
        XCTAssertEqual(options.ocr.dpi, 300)
        XCTAssertTrue(options.ocr.forceOCR)
        XCTAssertTrue(options.ocr.includeBBox)
        XCTAssertEqual(options.ocr.timeout, 2.5)
        XCTAssertEqual(options.ocr.targetPage, 3)
    }

    func testHelpAndVersion() throws {
        XCTAssertEqual(try CommandLineParser.parse(["-h"]), .help)
        XCTAssertEqual(try CommandLineParser.parse(["a.pdf", "--help"]), .help)
        XCTAssertEqual(try CommandLineParser.parse(["--version"]), .version)
    }

    func testDoubleDashTreatsRemainingArgumentsAsInputs() throws {
        XCTAssertEqual(try runOptions(["--", "--odd-name.pdf"]).inputs, ["--odd-name.pdf"])
    }

    func testRejectsInvalidArguments() {
        let invalid: [[String]] = [
            [],
            ["--bbox"],
            ["a.pdf", "--format", "xml"],
            ["a.pdf", "--format"],
            ["a.pdf", "--dpi", "50"],
            ["a.pdf", "--dpi", "601"],
            ["a.pdf", "--dpi", "high"],
            ["a.pdf", "--page", "0"],
            ["a.pdf", "--page", "1.5"],
            ["a.pdf", "--timeout", "0"],
            ["a.pdf", "--timeout", "inf"],
            ["a.pdf", "--lang", " , "],
            ["a.pdf", "--unknown"],
        ]
        for arguments in invalid {
            XCTAssertThrowsError(try CommandLineParser.parse(arguments), "\(arguments)") { error in
                XCTAssertEqual((error as? OCRError)?.exitCode, 1, "\(arguments)")
            }
        }
    }

    private func runOptions(_ arguments: [String]) throws -> RunOptions {
        guard case .run(let options) = try CommandLineParser.parse(arguments) else {
            throw XCTSkip("expected .run for \(arguments)")
        }
        return options
    }
}
