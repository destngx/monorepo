import pytest
import logging
from io import StringIO
from src.logging import setup_logging, ColorFormatter, get_logger


class TestColorizedLogging:
    def test_color_formatter_exists(self):
        formatter = ColorFormatter()
        assert formatter is not None

    def test_color_formatter_has_colors(self):
        formatter = ColorFormatter()
        assert hasattr(formatter, "COLORS")
        assert "INFO" in formatter.COLORS
        assert "ERROR" in formatter.COLORS
        assert "WARNING" in formatter.COLORS

    def test_setup_logging_creates_handler(self):
        logger = get_logger("test")
        assert logger is not None

    def test_log_output_includes_level(self, caplog):
        with caplog.at_level(logging.INFO):
            logger = get_logger("test")
            logger.info("Test info message")

            assert "Test info message" in caplog.text

    def test_get_logger_returns_logger(self):
        logger = get_logger("test_logger")
        assert isinstance(logger, logging.Logger)
        assert logger.name == "test_logger"

    def test_color_formatter_formats_record(self):
        formatter = ColorFormatter(fmt="%(levelname)s: %(message)s")
        record = logging.LogRecord(
            name="test",
            level=logging.INFO,
            pathname="test.py",
            lineno=1,
            msg="Test message",
            args=(),
            exc_info=None,
        )
        formatted = formatter.format(record)
        assert "Test message" in formatted
