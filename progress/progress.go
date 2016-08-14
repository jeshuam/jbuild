package progress

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/fatih/color"
	"github.com/sethgrid/curse"
)

var (
	// The main, global list of progress bars.
	progressBars              = []*ProgressBar{}
	progressBarUpdate         = make(chan *ProgressBar)
	progressBarUpdateFunction = &sync.WaitGroup{}
	disabled                  = false
)

func Start() {
	progressBarUpdateFunction.Add(1)
	go func() {
		term, _ := curse.New()
		doUpdate := func(update *ProgressBar) {
			id := len(progressBars) - update.id
			term.MoveUp(id)
			width, _, _ := curse.GetScreenDimensions()
			update.Display(term, width)
			term.MoveUp(1)
			term.MoveDown(id)
			update.lastUpdate = time.Now()
		}

		oldLen := len(progressBars)
		for update := range progressBarUpdate {
			if oldLen != len(progressBars) {
				for i := 0; i < len(progressBars)-oldLen; i++ {
					fmt.Println()
				}
			}

			oldLen = len(progressBars)
			doUpdate(update)
		}

		progressBarUpdateFunction.Done()
	}()
}

func Disable() {
	disabled = true
	progressBarUpdateFunction.Add(1)
	go func() {
		for range progressBarUpdate {

		}
		progressBarUpdateFunction.Done()
	}()
}

type progressBarDisplay struct {
	Name, Info, Prefix, Bar, Suffix, Tail string
}

func (pd *progressBarDisplay) Print(term *curse.Cursor, p *ProgressBar, name, info, prefix, bar, suffix, tail string) {
	// Save some color functions.
	yellow := color.New(color.FgYellow, color.Bold).SprintFunc()
	green := color.New(color.FgGreen, color.Bold).SprintFunc()
	grey := color.New(color.FgBlack, color.Bold).SprintFunc()

	// Set the colors of various parts.
	tailColor := grey
	nameColor := yellow
	if p.finished {
		nameColor = green
	}

	noColor := color.New().SprintFunc()

	printParts := func(newParts []string, colors []func(...interface{}) string) {
		// term.EraseCurrentLine()
		for i, newPart := range newParts {
			color := colors[i]

			fmt.Print(color(newPart))
		}

		fmt.Println()
	}

	printParts(
		[]string{name, info, prefix, bar, suffix, tail},
		[]func(...interface{}) string{nameColor, noColor, noColor, noColor, noColor, tailColor},
	)

	pd.Name = name
	pd.Info = info
	pd.Prefix = prefix
	pd.Bar = bar
	pd.Suffix = suffix
	pd.Tail = tail
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
	p.completeOps = p.totalOps
	p.lock.Unlock()

	progressBarUpdate <- p
}

func (p *ProgressBar) IsFinished() bool {
	return p.finished
}

func (p *ProgressBar) Display(term *curse.Cursor, width int) {
	// Add the header.
	name := fmt.Sprintf("%s", p.name)
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
	prefix := " ["
	suffix := "]"

	// Get the suffix.
	tail := fmt.Sprintf(" %s", p.suffix)

	bar := ""
	barLength := width - len(prefix) - len(suffix) - len(name) - len(info) - len(tail)
	for i := 0; i < barLength; i++ {
		percentComplete := (float64(i) / float64(barLength)) * 100
		if percentComplete < p.PercentComplete() {
			bar += "="
		} else {
			bar += " "
		}
	}

	bar = strings.Replace(bar, "= ", "=>", 1)

	// Set the colors of various parts.
	p.display.Print(term, p, name, info, prefix, bar, suffix, tail)
}

// Add a progress bar to the group and return a pointer to it
func AddBar(ops int, name string) *ProgressBar {
	progressBar := &ProgressBar{
		id:          len(progressBars),
		totalOps:    ops,
		completeOps: 0,
		name:        name,
	}

	if len(progressBars) == 0 {
		fmt.Println("\n")
	}
	progressBars = append(progressBars, progressBar)
	return progressBar
}

func Finish() {
	for len(progressBarUpdate) > 0 {

	}

	close(progressBarUpdate)
	progressBarUpdateFunction.Wait()
}
