package progress

import (
	"fmt"
	"strings"
	"sync"

	"github.com/fatih/color"
	"github.com/sethgrid/curse"
)

var (
	// The main, global list of progress bars.
	progressBars              = []*ProgressBar{}
	progressBarUpdate         = make(chan *ProgressBar)
	progressBarUpdateFunction = &sync.WaitGroup{}
)

func init() {
	progressBarUpdateFunction.Add(1)
	go func() {
		term, _ := curse.New()
		for update := range progressBarUpdate {
			// Move to this line.
			term.MoveDown(update.id)
			term.EraseCurrentLine()
			width, _, _ := curse.GetScreenDimensions()
			fmt.Println(update.Display(width))
			term.MoveUp(update.id + 1)
		}

		// Finish.
		term.MoveDown(len(progressBars) + 1)
		progressBarUpdateFunction.Done()
	}()
}

type ProgressBar struct {
	id          int        // The location of this progress bar within the list of bars.
	totalOps    int        // The total number of operations in the progress bar.
	completeOps int        // The number of completed operations.
	name        string     // The name of the progress bar.
	operation   string     // The current operation.
	suffix      string     // a suffix
	finished    bool       // True if this progress bar has finished.
	lock        sync.Mutex // A lock to ensure access to a progress bar is atomic.
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

	progressBarUpdate <- p
}

// Return the percentage complete in the range [0, 100].
func (p *ProgressBar) PercentComplete() float64 {
	return (float64(p.completeOps) / float64(p.totalOps)) * 100.0
}

func (p *ProgressBar) SetOperation(newOperation string) {
	p.lock.Lock()
	p.operation = newOperation
	p.lock.Unlock()

	progressBarUpdate <- p
}

func (p *ProgressBar) SetSuffix(newSuffix string) {
	p.lock.Lock()
	p.suffix = newSuffix
	p.lock.Unlock()

	progressBarUpdate <- p
}

func (p *ProgressBar) Finish() {
	p.lock.Lock()
	p.finished = true
	p.lock.Unlock()

	progressBarUpdate <- p
}

func (p *ProgressBar) IsFinished() bool {
	return p.finished
}

func (p *ProgressBar) Display(width int) string {
	// Add the header.
	header := fmt.Sprintf("%s", p.name)
	info := ""
	if !p.finished && len(p.operation) > 0 {
		info = fmt.Sprintf(" [%s...]", p.operation)
	}

	if p.finished {
		info = " [done]"
	}

	// Add the percent complete to the info.
	info += fmt.Sprintf(" %3.0f%%", p.PercentComplete())

	// Print the rest of the progress bar.
	progressHeader := " ["
	progressFooter := "]"

	// Get the suffix.
	suffix := fmt.Sprintf(" %s", p.suffix)

	progress := ""
	progressLength := width - len(progressHeader) - len(progressFooter) - len(header) - len(info) - len(suffix)
	for i := 0; i < progressLength; i++ {
		percentComplete := (float64(i) / float64(progressLength)) * 100
		if percentComplete < p.PercentComplete() {
			progress += "="
		} else {
			progress += " "
		}
	}

	progress = strings.Replace(progress, "= ", "=>", 1)

	// Set the colors of various parts.
	headerColor := color.New(color.FgYellow, color.Bold).SprintFunc()
	if p.finished {
		headerColor = color.New(color.FgGreen, color.Bold).SprintFunc()
	}

	suffixColor := color.New(color.FgBlack, color.Bold).SprintFunc()

	return headerColor(header) + info + progressHeader + progress + progressFooter + suffixColor(suffix)
}

// Add a progress bar to the group and return a pointer to it
func AddBar(ops int, name string) *ProgressBar {
	progressBar := &ProgressBar{
		id:          len(progressBars) + 1,
		totalOps:    ops,
		completeOps: 0,
		name:        name,
	}

	progressBars = append(progressBars, progressBar)
	fmt.Println()
	return progressBar
}

func Finish() {
	close(progressBarUpdate)
	progressBarUpdateFunction.Wait()
}
