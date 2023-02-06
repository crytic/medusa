package utils
import (
  "fmt"
  "golang.org/x/exp/slog"
)

// using any type for print args because that's what Printf uses, see:
// https://cs.opensource.google/go/go/+/master:src/fmt/print.go;l=232

type Logger struct {
  Level string "info"
}

func (l *Logger) GetLevel() string {
  return l.Level
}

func NewLogger(level string) *Logger {
  return &Logger{
    Level: level,
  }
}

func Debug(baseMsg string, args ...any) {
	slog.Debug(fmt.Sprintf(baseMsg, args...))
}

/*
func Info(baseMsg string, args ...any) {
	slog.Info(fmt.Sprintf(baseMsg, args))
}

func Warn(baseMsg string, args ...any) {
	slog.Warn(fmt.Sprintf(baseMsg, args))
}

func Error(baseMsg string, args ...any) {
	slog.Error(fmt.Sprintf(baseMsg, args))
}
*/
