// Package logging configures the process-wide slog logger.
package logging

import (
	"log/slog"
	"os"
	"strings"
)

// Configure installs a structured JSON logger that writes one object per line
// to stdout. Railway (and other log collectors) can parse the "time", "level",
// "msg", and attribute fields out of that format; the default logger writes
// plain text to stderr, which shows up as unlabelled noise.
//
// Set LOG_LEVEL to debug, info, warn, or error to change the minimum level.
func Configure() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: level(),
		// Railway keys off "level" (lowercase) and "message". slog emits
		// "level":"INFO" and "msg", which Railway normalizes anyway, but
		// writing the expected shape directly keeps the severity explicit.
		ReplaceAttr: func(groups []string, attr slog.Attr) slog.Attr {
			if len(groups) > 0 {
				return attr
			}
			switch attr.Key {
			case slog.LevelKey:
				return slog.String(slog.LevelKey, strings.ToLower(attr.Value.String()))
			case slog.MessageKey:
				return slog.String("message", attr.Value.String())
			}
			return attr
		},
	})))
}

func level() slog.Level {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("LOG_LEVEL"))) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
