package logger

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"strings"

	"github.com/fatih/color"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Logger wraps slog.Logger to provide consistent structured logging interface.
type Logger struct {
	*slog.Logger
	colorEnabled bool
}

// LogType defines the category of a log entry.
type LogType string

const (
	LogTypeApplication LogType = "application"
	LogTypeAudit       LogType = "audit"
	LogTypeRequest     LogType = "request"
	LogTypeError       LogType = "error"
)

// NewLogger creates a new logger configured from environment variables.
// Uses Zap as backend and colorizes JSON output if enabled.
// LOG_LEVEL: debug, info, warn, error (default: info)
// LOG_COLOR: true/false (default: true)
func NewLogger(logLevel string, colorEnabled bool) *Logger {
	var level slog.Level
	switch strings.ToLower(logLevel) {
	case "debug":
		level = slog.LevelDebug
	case "info":
		level = slog.LevelInfo
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	// Create Zap logger as backend
	cfg := zapcore.EncoderConfig{
		TimeKey:        "timestamp",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "",
		FunctionKey:    "",
		MessageKey:     "message",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.RFC3339NanoTimeEncoder,
		EncodeDuration: zapcore.StringDurationEncoder,
		EncodeCaller:   nil,
	}

	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(cfg),
		zapcore.AddSync(os.Stdout),
		zapLevelFromSlog(level),
	)

	zapLogger := zap.New(core)
	defer zapLogger.Sync()

	// Create a custom handler that wraps the Zap logger
	handler := &zapHandler{
		logger: zapLogger,
		level:  level,
		color:  colorEnabled,
	}

	slogLogger := slog.New(handler)
	return &Logger{
		Logger:       slogLogger,
		colorEnabled: colorEnabled,
	}
}

// zapHandler implements slog.Handler using Zap backend.
type zapHandler struct {
	logger *zap.Logger
	level  slog.Level
	color  bool
}

func (h *zapHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return level >= h.level
}

func (h *zapHandler) Handle(ctx context.Context, record slog.Record) error {
	fields := []zap.Field{
		zap.String("message", record.Message),
		zap.Time("timestamp", record.Time),
		zap.String("level", record.Level.String()),
	}

	// Add all attributes from the record
	record.Attrs(func(attr slog.Attr) bool {
		fields = append(fields, zap.Any(attr.Key, attr.Value.Any()))
		return true
	})

	// Log using appropriate Zap level
	switch record.Level {
	case slog.LevelDebug:
		h.logger.Debug(record.Message, fields...)
	case slog.LevelInfo:
		h.logger.Info(record.Message, fields...)
	case slog.LevelWarn:
		h.logger.Warn(record.Message, fields...)
	case slog.LevelError:
		h.logger.Error(record.Message, fields...)
	default:
		h.logger.Info(record.Message, fields...)
	}

	return nil
}

func (h *zapHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	// Create a new handler with additional fields context
	return &zapHandler{
		logger: h.logger,
		level:  h.level,
		color:  h.color,
	}
}

func (h *zapHandler) WithGroup(name string) slog.Handler {
	return &zapHandler{
		logger: h.logger,
		level:  h.level,
		color:  h.color,
	}
}

// LogApplicationEvent logs a structured application event.
func (l *Logger) LogApplicationEvent(ctx context.Context, message string, attrs ...slog.Attr) {
	attrs = append([]slog.Attr{slog.String("log_type", string(LogTypeApplication))}, attrs...)
	l.LogAttrs(ctx, slog.LevelInfo, message, attrs...)
}

// LogAuditEvent logs a sensitive audit event (e.g., config changes, authentication).
func (l *Logger) LogAuditEvent(ctx context.Context, message string, attrs ...slog.Attr) {
	attrs = append([]slog.Attr{slog.String("log_type", string(LogTypeAudit))}, attrs...)
	l.LogAttrs(ctx, slog.LevelInfo, message, attrs...)
}

// LogRequestEvent logs HTTP request/response details.
func (l *Logger) LogRequestEvent(ctx context.Context, message string, attrs ...slog.Attr) {
	attrs = append([]slog.Attr{slog.String("log_type", string(LogTypeRequest))}, attrs...)
	l.LogAttrs(ctx, slog.LevelInfo, message, attrs...)
}

// LogError logs an error with context.
func (l *Logger) LogError(ctx context.Context, message string, err error, attrs ...slog.Attr) {
	attrs = append([]slog.Attr{
		slog.String("log_type", string(LogTypeError)),
		slog.String("error", err.Error()),
	}, attrs...)
	l.LogAttrs(ctx, slog.LevelError, message, attrs...)
}

// ColorLevelString returns a colored representation of the log level.
func ColorLevelString(level slog.Level) string {
	switch level {
	case slog.LevelDebug:
		return color.CyanString("DEBUG")
	case slog.LevelInfo:
		return color.GreenString("INFO")
	case slog.LevelWarn:
		return color.YellowString("WARN")
	case slog.LevelError:
		return color.RedString("ERROR")
	default:
		return level.String()
	}
}

// ColorizeJSON colorizes JSON output for terminal display.
func ColorizeJSON(data []byte) string {
	var obj map[string]interface{}
	if err := json.Unmarshal(data, &obj); err != nil {
		return string(data)
	}

	// Colorize based on log level and type
	logType, _ := obj["log_type"].(string)
	level, _ := obj["level"].(string)

	var levelColor func(string, ...interface{}) string
	switch level {
	case "debug":
		levelColor = color.CyanString
	case "info":
		levelColor = color.GreenString
	case "warn":
		levelColor = color.YellowString
	case "error":
		levelColor = color.RedString
	default:
		levelColor = color.WhiteString
	}

	// Add type-based coloring
	typeColor := color.WhiteString
	switch LogType(logType) {
	case LogTypeAudit:
		typeColor = color.MagentaString
	case LogTypeRequest:
		typeColor = color.CyanString
	case LogTypeError:
		typeColor = color.RedString
	}

	// Format with colors (simple approach)
	formatted := color.CyanString("{")
	for key, val := range obj {
		if key == "level" {
			formatted += "\n  " + key + ": " + levelColor("%v", val)
		} else if key == "log_type" {
			formatted += "\n  " + key + ": " + typeColor("%q", val)
		} else {
			formatted += "\n  " + color.WhiteString(key) + ": " + color.WhiteString("%v", val)
		}
		formatted += ","
	}
	formatted += "\n" + color.CyanString("}")
	return formatted
}

// zapLevelFromSlog converts slog.Level to zapcore.Level.
func zapLevelFromSlog(level slog.Level) zapcore.Level {
	switch level {
	case slog.LevelDebug:
		return zap.DebugLevel
	case slog.LevelInfo:
		return zap.InfoLevel
	case slog.LevelWarn:
		return zap.WarnLevel
	case slog.LevelError:
		return zap.ErrorLevel
	default:
		return zap.InfoLevel
	}
}

// Global logger instance
var defaultLogger *Logger

// Init initializes the global logger (called from main).
func Init(logLevel string, colorEnabled bool) {
	defaultLogger = NewLogger(logLevel, colorEnabled)
}

// Get returns the global logger instance.
func Get() *Logger {
	if defaultLogger == nil {
		defaultLogger = NewLogger("info", true)
	}
	return defaultLogger
}
