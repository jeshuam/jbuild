package command

import (
	"encoding/gob"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/jeshuam/jbuild/common"
	"github.com/jeshuam/jbuild/config2/interfaces"
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
	TargetSpec string
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

func loadTestResult(target interfaces.TargetSpec) *testResult {
	// Don't load anything if the file doesn't exist.
	cacheFileName := filepath.Join(target.OutputPath(), target.Name()) + ".result"
	if !common.FileExists(cacheFileName) {
		return nil
	}

	// Check to see if the test executable was changed since the cache was made.
	cacheStat, _ := os.Stat(cacheFileName)
	outputStat, _ := os.Stat(filepath.Join(target.OutputPath(), target.Name()))
	if outputStat != nil && outputStat.ModTime().After(cacheStat.ModTime()) {
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

func runTest(target interfaces.TargetSpec, results chan testResult) {
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
	cmd := exec.Command(filepath.Join(target.OutputPath(), target.Name()))
	concurrentTestSemaphore <- true
	common.RunCommand(cmd, nil, func(output string, success bool, d time.Duration) {
		result := testResult{filepath.Join(target.OutputPath(), target.Name()), target.String(), success, output, d, false}
		result.save()
		results <- result
		<-concurrentTestSemaphore
	})
}

func runTests(targetsToTest map[string]interfaces.TargetSpec) chan testResult {
	rawResults := make(chan testResult)
	for target := range targetsToTest {
		for i := 0; i < int(*testRuns); i++ {
			go runTest(targetsToTest[target], rawResults)
		}
	}

	return rawResults
}

func collateTestResults(rawResults chan testResult, testRuns int) map[string][]testResult {
	results := make(map[string][]testResult, 0)
	for i := 0; i < testRuns; i++ {
		result := <-rawResults
		resultSpec := result.TargetSpec
		results[resultSpec] = append(results[resultSpec], result)
	}

	return results
}

func displayResultsForTarget(target string, results []testResult) {
	var (
		nPasses, nFails int
		totalDuration   time.Duration
		cached          bool

		gPrint = color.New(color.FgHiGreen, color.Bold).SprintfFunc()
		rPrint = color.New(color.FgHiRed, color.Bold).SprintfFunc()
	)

	// Aggregate the results.
	for _, result := range results {
		totalDuration += result.Duration
		cached = result.Cached

		if result.Passed {
			nPasses++
		} else {
			nFails++
		}
	}

	// Find the average duration.
	averageDuration := totalDuration / time.Duration(len(results))

	// Display the results.
	var state string
	cPrint := gPrint
	if nFails > 0 {
		state = "FAILED"
		cPrint = rPrint
	} else {
		state = "PASSED"
	}

	msg := fmt.Sprintf("%s in %s", state, averageDuration)
	if cached {
		msg += " (cached)"
	}

	if len(results) > 1 {
		msg += " (mean)"
		if nFails > 0 {
			msg += fmt.Sprintf(", %d/%d runs failed", nFails, len(results))
		}
	}

	fmt.Printf("\t%s: %s\n", cPrint(msg), target)

	// Decide if we should display the test output. We never display the output
	// for multi-run tests (this could probably be changed), and we don't usually
	// show test output for passed tests.
	barrier := strings.Repeat("=", 80)
	if len(results) == 1 {
		if (nFails > 0 && *testOutput != "none") || (nFails == 0 && *testOutput == "all") {
			fmt.Printf("\n%s\n%s%s\n\n", barrier, results[0].Output, barrier)
		}
	}
}

func RunTests(targetsToTest map[string]interfaces.TargetSpec) {
	// Run the tests once, for each command, and collect the results.
	rawResults := runTests(targetsToTest)

	// Collect all of the results into a result map, mapping target names to a
	// list of results.
	results := collateTestResults(rawResults, len(targetsToTest)*int(*testRuns))

	// Display the results to the screen.
	resultKeySorted := make([]string, 0, len(results))
	for resultKey := range results {
		resultKeySorted = append(resultKeySorted, resultKey)
	}

	sort.Strings(resultKeySorted)

	for _, target := range resultKeySorted {
		targetResults := results[target]
		displayResultsForTarget(target, targetResults)
	}
}
