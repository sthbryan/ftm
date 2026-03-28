package process

import (
	"strings"
	"sync"
)

type LogBuffer struct {
	mu        sync.RWMutex
	lines     []string
	maxLen    int
	OnNewLine func(string)
}

func NewLogBuffer() *LogBuffer {
	return &LogBuffer{
		lines:  make([]string, 0, 100),
		maxLen: 500,
	}
}

func (lb *LogBuffer) Write(p []byte) (n int, err error) {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	lines := strings.Split(string(p), "\n")
	for _, line := range lines {
		if line = strings.TrimSpace(line); line != "" {
			lb.lines = append(lb.lines, line)

			if lb.OnNewLine != nil {
				go lb.OnNewLine(line)
			}
		}
	}

	if len(lb.lines) > lb.maxLen {
		lb.lines = lb.lines[len(lb.lines)-lb.maxLen:]
	}

	return len(p), nil
}

func (lb *LogBuffer) GetLines() []string {
	lb.mu.RLock()
	defer lb.mu.RUnlock()

	result := make([]string, len(lb.lines))
	copy(result, lb.lines)
	return result
}
