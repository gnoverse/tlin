package lint

import (
	"fmt"
	"sync"

	"github.com/schollz/progressbar/v3"
)

// NewTUIObserver returns a ProgressObserver that renders a progress
// bar plus a sliding window of the most recently processed files.
func NewTUIObserver(label string) ProgressObserver {
	return &tuiObserver{
		label:       label,
		recentFiles: make([]string, maxShowRecentFiles),
	}
}

type tuiObserver struct {
	label       string
	bar         *progressbar.ProgressBar
	recentFiles []string
	mu          sync.Mutex
}

func (o *tuiObserver) OnStart(total int) {
	// Reserve vertical space for the recent-files window before
	// drawing the bar, then move the cursor back so the bar prints
	// above the window.
	for range maxShowRecentFiles + 1 {
		fmt.Println()
	}
	fmt.Printf("\033[%dA", maxShowRecentFiles+1)

	o.bar = progressbar.NewOptions(total,
		progressbar.OptionSetDescription(o.label),
		progressbar.OptionEnableColorCodes(true),
		progressbar.OptionSetWidth(40),
		progressbar.OptionShowCount(),
		progressbar.OptionSetTheme(progressbar.Theme{
			Saucer:        "[green]=[reset]",
			SaucerHead:    "[green]>[reset]",
			SaucerPadding: " ",
			BarStart:      "[",
			BarEnd:        "]",
		}))
}

func (o *tuiObserver) OnFile(name string) {
	o.mu.Lock()
	defer o.mu.Unlock()

	copy(o.recentFiles[1:], o.recentFiles[:maxShowRecentFiles-1])
	o.recentFiles[0] = name

	fmt.Printf("\033[%dA", maxShowRecentFiles)
	for j := range o.recentFiles {
		if o.recentFiles[j] != "" {
			fmt.Printf("\033[2K\r%s\n", o.recentFiles[j])
		} else {
			fmt.Printf("\033[2K\r\n")
		}
	}

	if o.bar != nil {
		_ = o.bar.Add(1)
	}
}

func (o *tuiObserver) OnDone() {
	fmt.Println()
}
