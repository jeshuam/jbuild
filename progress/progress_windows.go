package progress

import (
	"fmt"
	"strings"
	"time"

	"github.com/fatih/color"
)

var (
	moveToStart   = "\x1b[80D"
	moveUpOneLine = "\x1b[1A"
	lastUpdate    time.Time
)

func doUpdate(progressBar *ProgressBar) string {
	fmt.Print(moveToStart)
	fmt.Print(moveUpOneLine)

	// Print enough spaces to fill up the line.
	fmt.Print(strings.Repeat(" ", 80))
	fmt.Print(moveToStart)

	completeOps := 0
	totalOps := 0
	for _, progressBar := range progressBars {
		completeOps += progressBar.completeOps
		totalOps += progressBar.totalOps
	}

	// Prepare the new bar.
	green := color.New(color.FgGreen, color.Bold).SprintfFunc()
	blue := color.New(color.FgYellow, color.Bold).SprintfFunc()
	header := green("[%d/%d]", completeOps, totalOps)
	target := blue("%s", progressBar.name)

	// Make the new bar string and print it.
	update := fmt.Sprintf("%s %s: %s\n", header, target, progressBar.suffix)
	fmt.Print(update)
	return update
}

func Start() {
	progressBarUpdateFunction.Add(1)

	go func() {
		fmt.Println()

		// For each progress bar update...
		lastLineLength := 0
		for progressBar := range progressBarUpdate {
			// Ignore this update if they are coming in too fast.
			if time.Since(lastUpdate) < 50*time.Millisecond {
				continue
			}

			update := doUpdate(progressBar)

			// Print enough spaces to clear the previous line.
			if len(update) < lastLineLength {
				fmt.Print(strings.Repeat(" ", lastLineLength-len(update)))
			}

			lastLineLength = len(update)
			lastUpdate = time.Now()
		}

		// Finish the function.
		doUpdate(progressBars[0])
		progressBarUpdateFunction.Done()
	}()
}
