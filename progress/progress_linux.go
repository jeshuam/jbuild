package progress

import (
	"fmt"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/sethgrid/curse"
)

func StartComplex() {
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
			if len(progressBars) == 0 {
				fmt.Println()
				fmt.Println()
			}

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
