package ig

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
)

func textLogger(level slog.Level) (*slog.Logger, *bytes.Buffer) {
	var buf bytes.Buffer
	h := slog.NewTextHandler(&buf, &slog.HandlerOptions{
		Level: level,
		ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
			if a.Key == slog.TimeKey {
				return slog.Attr{} // stable output
			}
			return a
		},
	})
	return slog.New(h), &buf
}

func TestGadgetLoggerBridgesToSlog(t *testing.T) {
	l, buf := textLogger(slog.LevelInfo)
	log := newLogger(l)
	log.Warnf("disk %d%% full", 90)
	log.Debug("dropped: below handler level")

	if got := buf.String(); got != "level=WARN msg=\"disk 90% full\"\n" {
		t.Errorf("got %q", got)
	}
	if log.GetLevel() != logrus.InfoLevel {
		t.Errorf("level = %v, want info (from the handler)", log.GetLevel())
	}
}

func TestGlobalLogrusBridgesToSlog(t *testing.T) {
	l, buf := textLogger(slog.LevelDebug)
	redirectGlobal(l)
	t.Cleanup(func() { redirectGlobal(slog.New(slog.DiscardHandler)) })

	logrus.WithField("image", "trace_exec").Warn("signature verification disabled")
	logrus.Trace("dropped: below handler level")

	want := "level=WARN msg=\"signature verification disabled\" image=trace_exec\n"
	if got := buf.String(); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
	if strings.Contains(buf.String(), "dropped") {
		t.Error("trace message should be filtered")
	}
}

func TestSlogLevel(t *testing.T) {
	tests := []struct {
		in   logrus.Level
		want slog.Level
	}{
		{logrus.PanicLevel, slog.LevelError},
		{logrus.FatalLevel, slog.LevelError},
		{logrus.ErrorLevel, slog.LevelError},
		{logrus.WarnLevel, slog.LevelWarn},
		{logrus.InfoLevel, slog.LevelInfo},
		{logrus.DebugLevel, slog.LevelDebug},
		{logrus.TraceLevel, slogTrace},
	}
	for _, tt := range tests {
		if got := slogLevel(tt.in); got != tt.want {
			t.Errorf("slogLevel(%v) = %v, want %v", tt.in, got, tt.want)
		}
	}
}

func TestLevelFor(t *testing.T) {
	tests := []struct {
		handler slog.Level
		want    logrus.Level
	}{
		{slogTrace, logrus.TraceLevel},
		{slog.LevelDebug, logrus.DebugLevel},
		{slog.LevelInfo, logrus.InfoLevel},
		{slog.LevelWarn, logrus.WarnLevel},
		{slog.LevelError, logrus.ErrorLevel},
	}
	for _, tt := range tests {
		l, _ := textLogger(tt.handler)
		if got := levelFor(l); got != tt.want {
			t.Errorf("levelFor(%v) = %v, want %v", tt.handler, got, tt.want)
		}
	}
}
