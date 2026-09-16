package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

// LogWriter 按日期和文件大小双重维度轮转日志文件
// 文件名格式: {prefix}.{YYYYMMDD}.{seq}
// 示例: lobby.20260916.1, lobby.20260916.2, lobby.20260917.1
type LogWriter struct {
	dir     string // 日志目录
	prefix  string // 文件名前缀
	maxSize int64  // 单文件最大字节数
	maxAge  int    // 旧日志保留天数

	mu      sync.Mutex
	file    *os.File
	curDate string // 当前日期 YYYYMMDD
	curSize int64  // 当前文件已写字节数
	curSeq  int    // 当前文件序号
}

// NewLogWriter 创建日志写入器
// dir: 日志目录, prefix: 文件名前缀, maxSizeMB: 单文件最大 MB, maxAge: 保留天数
func NewLogWriter(dir, prefix string, maxSizeMB, maxAge int) (*LogWriter, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("create log dir error: %w", err)
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

	if err := w.openNew(); err != nil {
		return nil, err
	}
	return w, nil
}

func (w *LogWriter) Write(p []byte) (n int, err error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	today := time.Now().Format("20060102")

	// 日期变化：关闭旧文件，创建新文件
	if today != w.curDate {
		if w.file != nil {
			w.file.Close()
		}
		w.curDate = today
		w.curSeq = 0
		if err := w.openNew(); err != nil {
			return 0, err
		}
	}

	// 写入后超过大小限制：关闭旧文件，创建下一个序号文件
	if w.curSize+int64(len(p)) > w.maxSize {
		if w.file != nil {
			w.file.Close()
		}
		w.curSeq++
		if err := w.openNew(); err != nil {
			return 0, err
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

// openNew 创建新的日志文件，扫描目录确定序号，同时清理过期文件
func (w *LogWriter) openNew() error {
	// 扫描目录，找到当前日期的最大序号
	seq := w.findMaxSeq(w.curDate)
	if w.curSeq == 0 {
		w.curSeq = seq + 1
	}

	filename := filepath.Join(w.dir, fmt.Sprintf("%s.%s.%d", w.prefix, w.curDate, w.curSeq))
	f, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("open log file error: %w", err)
	}

	w.file = f
	info, _ := f.Stat()
	if info != nil {
		w.curSize = info.Size()
	} else {
		w.curSize = 0
	}

	// 触发旧文件清理
	w.cleanOldFiles()

	return nil
}

// findMaxSeq 查找指定日期下已存在的最大序号
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

// cleanOldFiles 清理超过 maxAge 天的日志文件
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

		// 解析日期部分: prefix.YYYYMMDD.seq
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
