package audit

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type Event struct {
	Time     time.Time     `json:"time"`
	User     string        `json:"user,omitempty"`
	IP       string        `json:"ip"`
	Method   string        `json:"method"`
	Path     string        `json:"path"`
	Status   int           `json:"status"`
	Bytes    int64         `json:"bytes"`
	Duration time.Duration `json:"durationNs"`
}

type Logger struct {
	directory string
	mu        sync.Mutex
}

func New(directory string) (*Logger, error) {
	if directory != "" {
		if err := os.MkdirAll(directory, 0o750); err != nil {
			return nil, err
		}
	}
	return &Logger{directory: directory}, nil
}

func (l *Logger) Write(event Event) {
	if l == nil || l.directory == "" {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	file, err := os.OpenFile(filepath.Join(l.directory, event.Time.Format("2006-01-02")+".jsonl"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o640)
	if err != nil {
		return
	}
	defer file.Close()
	_ = json.NewEncoder(file).Encode(event)
}
