package logger

import (
	"fmt"
	codeerror2 "lobby/model/codeerror"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

// LogWriter 按日期和文件大小双重维度轮转日志文件
type LogWriter struct {
	dir     string
	prefix  string
	maxSize int64
	maxAge  int

	mu      sync.Mutex
	file    *os.File
	curDate string
	curSize int64
	curSeq  int
}

// NewLogWriter 创建日志写入器
func NewLogWriter(dir, prefix string, maxSizeMB, maxAge int) (*LogWriter, *codeerror2.CodeError) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, codeerror2.LoggerError.Msg("create log dir error: " + err.Error())
	}

	if maxSizeMB <= 0 {
		maxSizeMB = 500
	}
	if maxAge <= 0 {
		maxAge = 30
	}

	w := &LogWriter{
		dir:     dir,
		prefix:  prefix,
		maxSize: int64(maxSizeMB) * 1024 * 1024,
		maxAge:  maxAge,
	}

	if ce := w.openNew(); ce != nil {
		return nil, ce
	}
	return w, nil
}

func (w *LogWriter) Write(p []byte) (n int, err error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	today := time.Now().Format("20060102")

	if today != w.curDate {
		if w.file != nil {
			w.file.Close()
		}
		w.curDate = today
		w.curSeq = 0
		if ce := w.openNew(); ce != nil {
			return 0, ce
		}
	}

	if w.curSize+int64(len(p)) > w.maxSize {
		if w.file != nil {
			w.file.Close()
		}
		w.curSeq++
		if ce := w.openNew(); ce != nil {
			return 0, ce
		}
	}

	n, err = w.file.Write(p)
	w.curSize += int64(n)
	return n, err
}

func (w *LogWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.file != nil {
		return w.file.Close()
	}
	return nil
}

func (w *LogWriter) openNew() *codeerror2.CodeError {
	seq := w.findMaxSeq(w.curDate)
	if w.curSeq == 0 {
		w.curSeq = seq + 1
	}

	filename := filepath.Join(w.dir, fmt.Sprintf("%s.%s.%d", w.prefix, w.curDate, w.curSeq))
	f, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return codeerror2.LoggerError.Msg("open log file error: " + err.Error())
	}

	w.file = f
	info, _ := f.Stat()
	if info != nil {
		w.curSize = info.Size()
	} else {
		w.curSize = 0
	}

	w.cleanOldFiles()
	return nil
}

func (w *LogWriter) findMaxSeq(date string) int {
	pattern := fmt.Sprintf("%s.%s.", w.prefix, date)
	maxSeq := 0

	entries, err := os.ReadDir(w.dir)
	if err != nil {
		return 0
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasPrefix(name, pattern) {
			continue
		}
		seqStr := name[len(pattern):]
		seq, err := strconv.Atoi(seqStr)
		if err != nil {
			continue
		}
		if seq > maxSeq {
			maxSeq = seq
		}
	}
	return maxSeq
}

func (w *LogWriter) cleanOldFiles() {
	if w.maxAge <= 0 {
		return
	}

	cutoff := time.Now().AddDate(0, 0, -w.maxAge).Format("20060102")

	entries, err := os.ReadDir(w.dir)
	if err != nil {
		return
	}

	prefix := w.prefix + "."
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasPrefix(name, prefix) {
			continue
		}

		rest := name[len(prefix):]
		parts := strings.SplitN(rest, ".", 2)
		if len(parts) < 2 {
			continue
		}
		date := parts[0]
		if len(date) != 8 {
			continue
		}
		if _, err := strconv.Atoi(date); err != nil {
			continue
		}

		if date < cutoff {
			os.Remove(filepath.Join(w.dir, name))
		}
	}
}
