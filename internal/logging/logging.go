package logging

import (
	"encoding/hex"
	"fmt"
	"github.com/sirupsen/logrus"
	"math/rand"
	"path/filepath"
	"runtime"
	"time"
)

type CustomFormatter struct {
	CustomFormatter logrus.Formatter
	TraceID         string
}

func (f *CustomFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	_, file, line, _ := runtime.Caller(7) // 6 will be logger.go, 7 will be the last caller
	entry.Message = fmt.Sprintf("%s:%d %s %s", filepath.Base(file), line, f.TraceID, entry.Message)
	return f.CustomFormatter.Format(entry)
}

func GenerateTraceID() string {
	length := 6
	bytes := make([]byte, length)
	rand.Seed(time.Now().UnixNano())
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}
