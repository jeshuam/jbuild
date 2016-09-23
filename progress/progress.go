package progress

import (
	"fmt"
	"sync"
	"time"
)

var (
	// The main, global list of progress bars.
	progressBars              = []*ProgressBar{}
	progressBarUpdate         = make(chan *ProgressBar)
	progressBarUpdateFunction = &sync.WaitGroup{}
	disabled                  = false
)

type progressBarDisplay struct {
	Name, Info, Prefix, Bar, Suffix, Tail string
}

type ProgressBar struct {
	id          int                // The location of this progress bar within the list of bars.
	totalOps    int                // The total number of operations in the progress bar.
	completeOps int                // The number of completed operations.
	name        string             // The name of the progress bar.
	operation   string             // The current operation.
	suffix      string             // a suffix
	finished    bool               // True if this progress bar has finished.
	lock        sync.Mutex         // A lock to ensure access to a progress bar is atomic.
	display     progressBarDisplay // The current display.
	lastUpdate  time.Time          // Last update to this progress bar.
}

// Increment the number of operations completed. This will cap the progress bar
// at whatever the maximum is.
func (p *ProgressBar) Increment() {
	p.lock.Lock()
	p.completeOps++
	if p.completeOps > p.totalOps {
		p.completeOps = p.totalOps
	}
	p.lock.Unlock()

	if !disabled {
		progressBarUpdate <- p
	}
}

// Return the percentage complete in the range [0, 100].
func (p *ProgressBar) PercentComplete() float64 {
	return (float64(p.completeOps) / float64(p.totalOps)) * 100.0
}

func (p *ProgressBar) SetOperation(newOperation string) {
	p.lock.Lock()
	p.operation = newOperation
	p.lock.Unlock()

	if !disabled {
		progressBarUpdate <- p
	}
}

func (p *ProgressBar) SetSuffix(newSuffix string) {
	p.lock.Lock()
	p.suffix = newSuffix
	p.lock.Unlock()

	if !disabled {
		progressBarUpdate <- p
	}
}

func (p *ProgressBar) Finish() {
	p.lock.Lock()
	p.finished = true
	p.completeOps = p.totalOps
	p.lock.Unlock()

	if !disabled {
		progressBarUpdate <- p
	}
}

func (p *ProgressBar) IsFinished() bool {
	return p.finished
}

// Add a progress bar to the group and return a pointer to it
func AddBar(ops int, name string) *ProgressBar {
	progressBar := &ProgressBar{
		id:          len(progressBars),
		totalOps:    ops,
		completeOps: 0,
		name:        name,
	}

	progressBars = append(progressBars, progressBar)
	return progressBar
}

func Finish() {
	for len(progressBarUpdate) > 0 {

	}

	close(progressBarUpdate)
	progressBarUpdateFunction.Wait()
	fmt.Printf("\n")
}

func Disable() {
	disabled = true
}
