package logger

import (
	"fmt"
	"io"
	"log"
	"os"

	"github.com/fatih/color"
	"github.com/mattn/go-colorable"
)

type LogLevel string

var (
	LogLevelDebug        LogLevel = "DEBUG"
	LogLevelPrint        LogLevel = "PRINT"
	LogLevelPrintSuccess LogLevel = "PRINT_SUCCESS"
	LogLevelInfo1        LogLevel = "INFO1"
	LogLevelInfo2        LogLevel = "INFO2"
	LogLevelWarn         LogLevel = "WARN"
)

type Logger struct {
	*log.Logger
}

var l *Logger

func New() *Logger {
	lgr := new(Logger)
	lgr.Logger = log.New(os.Stdout, "", 0)

	return lgr
}

func init() {
	l = New()
}

type NoColorWriter struct {
	io.Writer
}

func (lgr *Logger) setOut(outputFile string) error {
	if outputFile != "" {
		newWriter, err := os.OpenFile(outputFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil && !os.IsExist(err) {
			return err
		}

		ncw := colorable.NewNonColorable(newWriter)

		currentWriter := lgr.Logger.Writer()
		if currentWriter != nil {
			// combine with the current writer
			mw := io.MultiWriter(currentWriter, ncw)
			if err != nil {
				return err
			}

			lgr.Logger.SetOutput(mw)
		} else {
			lgr.Logger.SetOutput(newWriter)
		}

	}

	return nil
}

func SetOut(outputFile string) error {
	return l.setOut(outputFile)
}

func (lgr *Logger) print(level LogLevel, msg string, a ...any) {
	switch level {
	case LogLevelDebug:
		lgr.Println(fmt.Sprintf(msg, a...))
	case LogLevelPrint:
		lgr.Println(color.CyanString(msg, a...))
	case LogLevelPrintSuccess:
		lgr.Println(color.GreenString(msg, a...))
	case LogLevelInfo1:
		lgr.Println(color.BlueString(msg, a...))
	case LogLevelInfo2:
		lgr.Println(color.MagentaString(msg, a...))
	case LogLevelWarn:
		lgr.Println(color.YellowString(msg, a...))
	default:
		lgr.Println(fmt.Sprintf(msg, a...))
	}
}

func Debug(msg string, a ...any) {
	l.print(LogLevelDebug, msg, a...)
}

// print cyan
func Print(msg string, a ...any) {
	l.print(LogLevelPrint, msg, a...)
}

// print green
func PrintSuccess(msg string, a ...any) {
	l.print(LogLevelPrintSuccess, msg, a...)
}

// print blue
func Info1(msg string, a ...any) {
	l.print(LogLevelInfo1, msg, a...)
}

// print magenta
func Info2(msg string, a ...any) {
	l.print(LogLevelInfo2, msg, a...)
}

func Warn(msg string, a ...any) {
	l.print(LogLevelWarn, msg, a...)
}

func (lgr *Logger) error(msg string, a ...any) {
	lgr.Fatalln(color.RedString(msg, a...))
}

func Error(msg string, a ...any) {
	l.error(msg, a...)
}
