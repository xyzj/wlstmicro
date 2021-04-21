package wmv2

import (
	"fmt"
	"io"
	"strings"

	"github.com/xyzj/gopsu"
)

var (
	logReplaceDefault = strings.NewReplacer("\t", "", "\r", "", "\n", " ")
)

// WriteDebug debug日志
func (fw *WMFrameWorkV2) WriteDebug(name, msg string) {
	fw.WriteLog(name, msg, 10)
}

// WriteInfo Info日志
func (fw *WMFrameWorkV2) WriteInfo(name, msg string) {
	fw.WriteLog(name, msg, 20)
}

// WriteWarning Warning日志
func (fw *WMFrameWorkV2) WriteWarning(name, msg string) {
	fw.WriteLog(name, msg, 30)
}

// WriteError Error日志
func (fw *WMFrameWorkV2) WriteError(name, msg string) {
	fw.WriteLog(name, msg, 40)
}

// WriteSystem System日志
func (fw *WMFrameWorkV2) WriteSystem(name, msg string) {
	fw.WriteLog(name, msg, 90)
}

// WriteLog 写公共日志
// name： 日志类别，如sys，mq，db这种
// msg： 日志信息
// level： 日志级别10,20，30,40,90
func (fw *WMFrameWorkV2) WriteLog(name, msg string, level int) {
	if level <= 0 || level < *logLevel {
		return
	}
	if name != "" {
		name = "[" + name + "] "
	}
	msg = logReplaceDefault.Replace(msg)
	switch level {
	case 10:
		fw.wmLog.Debug(fmt.Sprintf("%s%s", name, msg))
	case 20:
		fw.wmLog.Info(fmt.Sprintf("%s%s", name, msg))
	case 30:
		fw.wmLog.Warning(fmt.Sprintf("%s%s", name, msg))
	case 40:
		fw.wmLog.Error(fmt.Sprintf("%s%s", name, msg))
	case 90:
		fw.wmLog.System(fmt.Sprintf("%s%s", name, msg))
	}
}

// StdLogger StdLogger
type StdLogger struct {
	Name        string
	LogReplacer *strings.Replacer
	LogWriter   gopsu.Logger
}

func (l *StdLogger) writeLog(name, msg string, level int) {
	if level <= 0 || level < *logLevel {
		return
	}
	if name != "" {
		name = "[" + name + "] "
	}
	if l.LogReplacer != nil {
		msg = l.LogReplacer.Replace(strings.TrimSpace(logReplaceDefault.Replace(msg)))
	}
	switch level {
	case 10:
		l.LogWriter.Debug(fmt.Sprintf("%s%s", name, msg))
	case 20:
		l.LogWriter.Info(fmt.Sprintf("%s%s", name, msg))
	case 30:
		l.LogWriter.Warning(fmt.Sprintf("%s%s", name, msg))
	case 40:
		l.LogWriter.Error(fmt.Sprintf("%s%s", name, msg))
	case 90:
		l.LogWriter.System(fmt.Sprintf("%s%s", name, msg))
	}
}

// Debug Debug
func (l *StdLogger) Debug(msgs string) {
	l.writeLog(l.Name, msgs, 10)
}

// Info Info
func (l *StdLogger) Info(msgs string) {
	l.writeLog(l.Name, msgs, 20)
}

// Warning Warn
func (l *StdLogger) Warning(msgs string) {
	l.writeLog(l.Name, msgs, 30)
}

// Error Error
func (l *StdLogger) Error(msgs string) {
	l.writeLog(l.Name, msgs, 40)
}

// System System
func (l *StdLogger) System(msgs string) {
	l.writeLog(l.Name, msgs, 90)
}

// DebugFormat Debug
func (l *StdLogger) DebugFormat(f string, msg ...interface{}) {
	if f == "" {
		l.writeLog(l.Name, fmt.Sprintf("%v", msg), 10)
	} else {
		l.writeLog(l.Name, fmt.Sprintf(f, msg...), 10)
	}
}

// InfoFormat Info
func (l *StdLogger) InfoFormat(f string, msg ...interface{}) {
	if f == "" {
		l.writeLog(l.Name, fmt.Sprintf("%v", msg), 20)
	} else {
		l.writeLog(l.Name, fmt.Sprintf(f, msg...), 20)
	}
}

// WarningFormat Warn
func (l *StdLogger) WarningFormat(f string, msg ...interface{}) {
	if f == "" {
		l.writeLog(l.Name, fmt.Sprintf("%v", msg), 30)
	} else {
		l.writeLog(l.Name, fmt.Sprintf(f, msg...), 30)
	}
}

// ErrorFormat Error
func (l *StdLogger) ErrorFormat(f string, msg ...interface{}) {
	if f == "" {
		l.writeLog(l.Name, fmt.Sprintf("%v", msg), 40)
	} else {
		l.writeLog(l.Name, fmt.Sprintf(f, msg...), 40)
	}
}

// SystemFormat System
func (l *StdLogger) SystemFormat(f string, msg ...interface{}) {
	if f == "" {
		l.writeLog(l.Name, fmt.Sprintf("%v", msg), 90)
	} else {
		l.writeLog(l.Name, fmt.Sprintf(f, msg...), 90)
	}
}

// DefaultWriter 返回默认writer
func (l *StdLogger) DefaultWriter() io.Writer {
	return l.LogWriter.DefaultWriter()
}
