package logging

import (
	"bytes"
	"fmt"
	"github.com/sirupsen/logrus"
	"runtime"
	"strings"
)

const (
	red    = 31
	yellow = 33
	blue   = 36
	gray   = 37
	green  = 32
)

// Formatter that is called on by logrus.
type RuniacFormatter struct {
	// DisableTimestamp allows disabling automatic timestamps in output
	DisableColors bool
}

func (f *RuniacFormatter) isColored() bool {
	isColored := runtime.GOOS != "windows"

	return isColored && !f.DisableColors
}

// Format the log entry. Implements logrus.Formatter.
func (f *RuniacFormatter) Format(entry *logrus.Entry) ([]byte, error) {

	var b *bytes.Buffer
	if entry.Buffer != nil {
		b = entry.Buffer
	} else {
		b = &bytes.Buffer{}
	}

	if f.isColored() {
		f.prependColored(b, entry.Level)
	} else {
		_, _ = fmt.Fprintf(b, "[%s] ", strings.ToUpper(entry.Level.String()))
	}

	if _, ok := entry.Data["action"]; ok {

		stepId := []string{}

		stepId = appendIfSet(stepId, entry, "track")
		stepId = appendIfSet(stepId, entry, "step")
		stepId = appendIfSet(stepId, entry, "regionDeployType")
		stepId = appendIfSet(stepId, entry, "region")

		_, _ = fmt.Fprintf(b, "(%s %s)   ", entry.Data["action"], strings.Join(stepId, "/"))
	}

	_, _ = fmt.Fprintf(b, "%s", entry.Message)

	if err, ok := entry.Data["error"]; ok {
		b.WriteString(fmt.Sprintf("   (%s)", err))
	}

	if f.isColored() {
		f.postpendColored(b)
	}

	b.WriteByte('\n')

	return b.Bytes(), nil
}

func appendIfSet(slice []string, entry *logrus.Entry, key string) []string {
	if _, ok := entry.Data[key]; ok {
		return append(slice, fmt.Sprintf("%v", entry.Data[key]))
	}

	return slice
}

func (f *RuniacFormatter) prependColored(b *bytes.Buffer, lvl logrus.Level) {
	var levelColor int
	switch lvl {
	case logrus.DebugLevel, logrus.TraceLevel:
		levelColor = gray
	case logrus.WarnLevel:
		levelColor = yellow
	case logrus.ErrorLevel, logrus.FatalLevel, logrus.PanicLevel:
		levelColor = red
	default:
		levelColor = green
	}

	fmt.Fprintf(b, "\x1b[%dm", levelColor)
}

func (f *RuniacFormatter) postpendColored(b *bytes.Buffer) {
	fmt.Fprint(b, "\x1b[0m")
}
