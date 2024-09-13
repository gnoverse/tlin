package internal

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	tt "github.com/gnoswap-labs/tlin/internal/types"
)

func (e *Engine) StartWatching() error {
	if e.isWatching {
		return fmt.Errorf("already watching")
	}

	for _, dir := range e.watchDirs {
		err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				return e.watcher.Add(path)
			}
			return nil
		})
		if err != nil {
			return fmt.Errorf("error adding directory to watcher: %w", err)
		}
	}

	e.isWatching = true
	go e.watchLoop()
	return nil
}

func (e *Engine) StopWatching() error {
	if !e.isWatching {
		log.Println("not watching")
	}

	e.isWatching = false
	return e.watcher.Close()
}

func (e *Engine) watchLoop() {
	for e.isWatching {
		select {
		case event, ok := <-e.watcher.Events:
			if !ok {
				return
			}
			e.handleFileEvent(event)
		case err, ok := <-e.watcher.Errors:
			if !ok {
				return
			}
			log.Printf("error: %v", err)
		}
	}
}

func (e *Engine) handleFileEvent(event fsnotify.Event) {
	if event.Op&fsnotify.Write == fsnotify.Write {
		// process file when detect change
		if strings.HasSuffix(event.Name, ".go") || strings.HasSuffix(event.Name, ".gno") || strings.HasSuffix(event.Name, ".mod") {
			// wait for a while after file change to consider multiple changes as one
			time.Sleep(100 * time.Millisecond)
			issue, err := e.Run(event.Name)
			if err != nil {
				log.Printf("error: %v", err)
				return
			}
			e.reportIssues(event.Name, issue)
		}
	}
}

func (e *Engine) reportIssues(filename string, issues []tt.Issue) {
	if len(issues) == 0 {
		log.Printf("no issues found in %s", filename)
	}

	log.Printf("found %d issues in %s", len(issues), filename)
	for _, issue := range issues {
		log.Printf("- %s: %s", issue.Rule, issue.Message)
	}
}
