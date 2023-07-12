package logging

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"path/filepath"
	"runtime"
)

type CustomFormatter struct {
	logrus.Formatter
}

func (f *CustomFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	_, file, line, _ := runtime.Caller(7) // 6 will be logger.go, 7 will be the last caller
	entry.Message = fmt.Sprintf("%s:%d %s", filepath.Base(file), line, entry.Message)
	return f.Formatter.Format(entry)
}
