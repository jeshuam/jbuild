package command

import (
	"encoding/gob"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"time"

	"github.com/fatih/color"
	"github.com/jeshuam/jbuild/common"
	"github.com/jeshuam/jbuild/config"
	"github.com/op/go-logging"
)

var (
	log = logging.MustGetLogger("jbuild")

	forceRunTests = flag.Bool("force_run_tests", false, "Whether or not we should for tests to run.")
	testRuns      = flag.Uint("test_runs", 1, "Number of times to run each test.")
	testOutput    = flag.String("test_output", "errors", "The verbosity of test output to show. Can be all|errors|none.")
	testThreads   = flag.Int("test_threads", runtime.NumCPU()/3, "Number of threads to use when running tests.")

	concurrentTestSemaphore = make(chan bool, *testThreads)
)

type testResult struct {
	TestBinary string
	TargetSpec config.TargetSpec
	Passed     bool
	Output     string
	Duration   time.Duration
	Cached     bool
}

func (this *testResult) save() {
	// Open the output cache file.
	cacheFileName := this.TestBinary + ".result"
	cacheFile, err := os.Create(cacheFileName)
	if err != nil {
		log.Errorf("Could not save test result cache: '%s': %v", cacheFileName, err)
	}

	// Encode the test result into this file.
	defer cacheFile.Close()
	encoder := gob.NewEncoder(cacheFile)
	err = encoder.Encode(*this)
	if err != nil {
		log.Errorf("Could not encode test result cache: %v", err)
	}
}

func loadTestResult(target *config.Target) *testResult {
	// Don't load anything if the file doesn't exist.
	cacheFileName := target.Output[0] + ".result"
	if !common.FileExists(cacheFileName) {
		return nil
	}

	// Try to open the file; we should fail if the file exists but is not
	// openable.
	cacheFile, err := os.Open(cacheFileName)
	if err != nil {
		log.Errorf("Could not load cached test result file '%s': %v", cacheFileName, err)
	}

	// If we can open the file, then decode the object in the file!
	defer cacheFile.Close()
	result := new(testResult)
	decoder := gob.NewDecoder(cacheFile)
	err = decoder.Decode(result)
	if err != nil {
		log.Errorf("Could not load cached test result file '%s': %v", cacheFileName, err)
	}

	// The result is definiely cached (we just loaded it).
	result.Cached = true

	return result
}

func runTest(target *config.Target, results chan testResult) {
	// If we aren't being forced to run tests, then try to load a cached test
	// result file.
	if !*forceRunTests && *testRuns == 1 {
		result := loadTestResult(target)
		if result != nil {
			results <- *result
			return
		}
	}

	// Either we are being forced to run tests, or this test has not been cached
	// recently. Run the test!
	cmd := exec.Command(target.Output[0])
	concurrentTestSemaphore <- true
	common.RunCommand(cmd, nil, func(output string, success bool, d time.Duration) {
		result := testResult{target.Output[0], *target.Spec, success, output, d, false}
		result.save()
		results <- result
		<-concurrentTestSemaphore
	})
}

func RunTests(targetsToTest []*config.Target) {
	// Run the tests once, for each command, and collect the results.
	rawResults := make(chan testResult)
	for _, target := range targetsToTest {
		for i := 0; i < int(*testRuns); i++ {
			go runTest(target, rawResults)
		}
	}

	// Collect all of the results into a result map, mapping target names to a
	// list of results.
	results := make(map[string][]testResult, 0)
	for i := 0; i < len(targetsToTest)*int(*testRuns); i++ {
		result := <-rawResults
		resultSpec := result.TargetSpec.String()
		results[resultSpec] = append(results[resultSpec], result)
	}

	// Display the results to the screen.
	gPrint := color.New(color.FgHiGreen, color.Bold).SprintfFunc()
	rPrint := color.New(color.FgHiRed, color.Bold).SprintfFunc()
	for target, targetResults := range results {
		var (
			nPasses, nFails int
			totalDuration   time.Duration
			cached          bool
		)

		// Aggregate the results.
		for _, targetResult := range targetResults {
			totalDuration += targetResult.Duration
			if targetResult.Cached {
				cached = true
			}

			if targetResult.Passed {
				nPasses++
			} else {
				nFails++
			}
		}

		// Find the average duration.
		averageDuration := totalDuration / time.Duration(len(targetResults))

		// Display the results.
		if nFails > 0 {
			msg := fmt.Sprintf("FAILED in %s", averageDuration)
			if len(targetResults) > 1 {
				msg += " (mean)"
			}

			if cached {
				msg += " (cached)"
			}

			if nFails > 1 {
				msg += fmt.Sprintf(", %d/%d runs failed", nFails, len(targetResults))
			}

			fmt.Printf("\t%s: %s\n", rPrint(msg), target)

			// If we failed and only did a single run, the display the result. We
			// don't want to display the results if there were multiple runs.
			if len(targetResults) == 1 && *testOutput != "none" {
				barrier := "================================================================="
				fmt.Printf("\n%s\n%s%s\n", barrier, targetResults[0].Output, barrier)
			}
		} else {
			msg := fmt.Sprintf("PASSED in %s", averageDuration)
			if len(targetResults) > 1 {
				msg += " (mean)"
			}

			if cached {
				msg += " (cached)"
			}

			fmt.Printf("\t%s: %s\n", gPrint(msg), target)
			if len(targetResults) == 1 && *testOutput == "all" {
				barrier := "================================================================="
				fmt.Printf("\n%s\n%s%s\n", barrier, targetResults[0].Output, barrier)
			}
		}
	}
}
