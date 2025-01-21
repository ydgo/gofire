package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"io"
	"os"
)

type Level = zapcore.Level

const (
	DebugLevel = zapcore.DebugLevel
	InfoLevel  = zapcore.InfoLevel
	WarnLevel  = zapcore.WarnLevel
	ErrorLevel = zapcore.ErrorLevel
)

type Logger struct {
	l  *zap.SugaredLogger
	al *zap.AtomicLevel
}

func New(out io.Writer, level Level) *Logger {
	al := zap.NewAtomicLevelAt(level)
	cfg := zap.NewProductionEncoderConfig()
	cfg.EncodeTime = zapcore.ISO8601TimeEncoder
	core := zapcore.NewCore(
		zapcore.NewConsoleEncoder(cfg),
		zapcore.AddSync(out),
		al)
	l := zap.New(core).WithOptions(zap.AddCaller(), zap.AddCallerSkip(1))
	return &Logger{
		l:  l.Sugar(),
		al: &al,
	}
}

func (l *Logger) Debug(args ...interface{}) {
	l.l.Debug(args...)
}

func (l *Logger) Info(args ...interface{}) {
	l.l.Info(args...)
}

func (l *Logger) Warn(args ...interface{}) {
	l.l.Warn(args...)
}

func (l *Logger) Error(args ...interface{}) {
	l.l.Error(args...)
}

func (l *Logger) Debugf(format string, args ...interface{}) {
	l.l.Debugf(format, args...)
}

func (l *Logger) Infof(format string, args ...interface{}) {
	l.l.Infof(format, args...)
}

func (l *Logger) Warnf(format string, args ...interface{}) {
	l.l.Warnf(format, args...)
}

func (l *Logger) Errorf(format string, args ...interface{}) {
	l.l.Errorf(format, args...)
}

func (l *Logger) Fatal(args ...interface{}) {
	l.l.Fatal(args...)
}

func (l *Logger) Fatalf(format string, args ...interface{}) {
	l.l.Fatalf(format, args...)
}

func (l *Logger) Sync() error {
	return l.l.Sync()
}

var std = New(os.Stdout, InfoLevel)

func Default() *Logger {
	return std
}

func ReplaceDefault(l *Logger) { std = l }

func Debug(args ...interface{}) { std.l.Debug(args...) }

func Debugf(format string, args ...interface{}) { std.l.Debugf(format, args...) }

func Info(args ...interface{}) { std.l.Info(args...) }

func Infoln(args ...interface{}) { std.l.Infoln(args...) }

func Infof(format string, args ...interface{}) { std.l.Infof(format, args...) }

func Warn(args ...interface{}) { std.l.Warn(args...) }

func Warnln(args ...interface{}) { std.l.Warnln(args...) }

func Warnf(format string, args ...interface{}) { std.l.Warnf(format, args...) }

func Error(args ...interface{}) { std.l.Error(args...) }

func Errorf(format string, args ...interface{}) { std.l.Errorf(format, args...) }

func Fatal(args ...interface{}) { std.l.Fatal(args...) }

func Fatalf(format string, args ...interface{}) { std.l.Fatalf(format, args...) }

func SetLevel(level Level) { std.al.SetLevel(level) }

func Sync() error { return std.l.Sync() }
