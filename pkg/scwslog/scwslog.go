package scwslog

import (
	"context"
	"fmt"
	"log/slog"

	scwlogger "github.com/scaleway/scaleway-sdk-go/logger"
)

type ScwSlogger struct {
	logger *slog.Logger
}

func NewLogger(logger *slog.Logger) *ScwSlogger {
	return &ScwSlogger{logger: logger}
}

// Debugf implements logger.Logger.
func (s *ScwSlogger) Debugf(format string, args ...any) {
	s.logger.Debug(fmt.Sprintf(format, args...))
}

// Errorf implements logger.Logger.
func (s *ScwSlogger) Errorf(format string, args ...any) {
	s.logger.Error(fmt.Sprintf(format, args...))
}

// Infof implements logger.Logger.
func (s *ScwSlogger) Infof(format string, args ...any) {
	s.logger.Info(fmt.Sprintf(format, args...))
}

// ShouldLog implements logger.Logger.
func (s *ScwSlogger) ShouldLog(level scwlogger.LogLevel) bool {
	return s.logger.Enabled(context.Background(), scwLogLevelToSlogLevel(level))
}

// Warningf implements logger.Logger.
func (s *ScwSlogger) Warningf(format string, args ...any) {
	s.logger.Warn(fmt.Sprintf(format, args...))
}

var _ scwlogger.Logger = (*ScwSlogger)(nil)

func scwLogLevelToSlogLevel(level scwlogger.LogLevel) slog.Level {
	switch level {
	case scwlogger.LogLevelDebug:
		return slog.LevelDebug
	case scwlogger.LogLevelInfo:
		return slog.LevelInfo
	case scwlogger.LogLevelWarning:
		return slog.LevelWarn
	case scwlogger.LogLevelError:
		return slog.LevelError
	}

	return slog.LevelInfo
}
