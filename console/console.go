package console

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"golang.org/x/term"
)

type Console struct {
	mu      sync.Mutex
	logs    []string
	status  string
	running bool
	rows    int
	cols    int
}

func New() *Console {
	c := &Console{
		logs:    make([]string, 0, 2000),
		running: true,
	}
	c.updateSize()

	fmt.Print("\033[2J\033[H")
	c.drawBanner()
	c.Log("✅ 就绪，等待插件连接...")

	go c.renderLoop()
	return c
}

func (c *Console) Log(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	c.mu.Lock()
	if len(c.logs) >= 2000 {
		c.logs = c.logs[1000:]
	}
	c.logs = append(c.logs, msg)
	c.mu.Unlock()
}

func (c *Console) SetStatus(s string) {
	c.mu.Lock()
	c.status = s
	c.mu.Unlock()
}

func (c *Console) Close() {
	c.mu.Lock()
	c.running = false
	c.mu.Unlock()
	c.render()
	fmt.Print("\n")
}

func (c *Console) renderLoop() {
	t := time.NewTicker(200 * time.Millisecond)
	defer t.Stop()
	for c.running {
		<-t.C
		c.render()
	}
}

func (c *Console) render() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.updateSize()
	fmt.Print("\033[H")

	logMax := c.rows - 3
	if logMax < 1 {
		logMax = 1
	}
	start := len(c.logs) - logMax
	if start < 0 {
		start = 0
	}

	logLines := len(c.logs) - start
	for i := start; i < len(c.logs); i++ {
		line := c.logs[i]
		if len(line) > c.cols {
			line = line[:c.cols]
		}
		fmt.Print("\033[K")
		fmt.Println(line)
	}

	for i := logMax - logLines; i > 0; i-- {
		fmt.Print("\033[K\n")
	}

	sep := strings.Repeat("─", max(c.cols, 1))
	fmt.Print("\033[K")
	fmt.Println(sep)

	status := c.status
	if len(status) > c.cols {
		status = status[:c.cols]
	}
	fmt.Print("\033[K")
	fmt.Print(status)
	fmt.Print("\033[K")
}

func (c *Console) drawBanner() {
	c.Log("")
	c.Log("    ██╗   ██╗ ██████╗ ██╗  ██╗")
	c.Log("    ██║   ██║██╔═══██╗╚██╗██╔╝")
	c.Log("    ██║   ██║██║   ██║ ╚███╔╝")
	c.Log("    ╚██╗ ██╔╝██║   ██║ ██╔██╗")
	c.Log("     ╚████╔╝ ╚██████╔╝██╔╝ ██╗")
	c.Log("      ╚═══╝   ╚═════╝ ╚═╝  ╚═╝")
	c.Log("")
	c.Log("     全新的AMLL Player 音频后端 v0.0.1")
	c.Log("     github.com/xiaowumin-mark/VoxBackend")
	c.Log("")
}

func (c *Console) updateSize() {
	fd := int(os.Stdout.Fd())
	if w, h, err := term.GetSize(fd); err == nil {
		c.cols = w
		c.rows = h
	} else {
		if c.cols == 0 {
			c.cols = 80
		}
		if c.rows == 0 {
			c.rows = 24
		}
	}
}
