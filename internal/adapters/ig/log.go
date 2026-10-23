package ig

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"sync/atomic"

	"github.com/sirupsen/logrus"

	"github.com/inspektor-gadget/inspektor-gadget/pkg/logger"
)

// IG logs through logrus in two ways: a logger.Logger passed to each gadget,
// and the global logrus logger. Both are bridged to slog.

// newLogger returns an IG logger writing to l.
func newLogger(l *slog.Logger) logger.Logger {
	s := &slogLogger{l: l}
	s.level.Store(uint32(levelFor(l)))
	return logger.NewFromGenericLogger(s)
}

// redirectGlobal sends logs from IG's global logrus logger to l.
func redirectGlobal(l *slog.Logger) {
	g := logrus.StandardLogger()
	g.SetOutput(io.Discard)
	g.SetLevel(levelFor(l))
	g.ReplaceHooks(logrus.LevelHooks{})
	g.AddHook(slogHook{l: l})
}

// slogLogger implements logger.GenericLoggerWithLevelSetter on slog.
type slogLogger struct {
	l     *slog.Logger
	level atomic.Uint32 // logrus.Level
}

func (s *slogLogger) Log(severity logger.Level, params ...any) {
	if severity <= s.GetLevel() {
		s.l.Log(context.Background(), slogLevel(severity), fmt.Sprint(params...))
	}
}

func (s *slogLogger) Logf(severity logger.Level, format string, params ...any) {
	if severity <= s.GetLevel() {
		s.l.Log(context.Background(), slogLevel(severity), fmt.Sprintf(format, params...))
	}
}

func (s *slogLogger) SetLevel(level logger.Level) { s.level.Store(uint32(level)) }

func (s *slogLogger) GetLevel() logger.Level { return logger.Level(s.level.Load()) }

// slogHook forwards logrus entries to slog.
type slogHook struct{ l *slog.Logger }

func (h slogHook) Levels() []logrus.Level { return logrus.AllLevels }

func (h slogHook) Fire(e *logrus.Entry) error {
	attrs := make([]any, 0, 2*len(e.Data))
	for k, v := range e.Data {
		attrs = append(attrs, k, v)
	}
	h.l.Log(context.Background(), slogLevel(e.Level), e.Message, attrs...)
	return nil
}

// slogTrace is below slog.LevelDebug, for logrus's trace level.
const slogTrace = slog.LevelDebug - 4

// slogLevel maps a logrus level to slog. Panic and fatal map to error: they
// are logged, not acted on.
func slogLevel(l logrus.Level) slog.Level {
	switch {
	case l <= logrus.ErrorLevel:
		return slog.LevelError
	case l == logrus.WarnLevel:
		return slog.LevelWarn
	case l == logrus.InfoLevel:
		return slog.LevelInfo
	case l == logrus.DebugLevel:
		return slog.LevelDebug
	default:
		return slogTrace
	}
}

// levelFor returns the most verbose logrus level that l's handler accepts, so
// IG doesn't format messages that would be dropped.
func levelFor(l *slog.Logger) logrus.Level {
	for _, lv := range []logrus.Level{logrus.TraceLevel, logrus.DebugLevel, logrus.InfoLevel, logrus.WarnLevel} {
		if l.Enabled(context.Background(), slogLevel(lv)) {
			return lv
		}
	}
	return logrus.ErrorLevel
}
