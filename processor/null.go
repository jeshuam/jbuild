package processor

import (
	"math/rand"
	"time"

	"github.com/jeshuam/jbuild/config"
)

// A null processor doesn't actually do anything when it processes targets
// except mark them as processed.
type NullProcessor struct {
}

func (p NullProcessor) Process(target *config.Target) error {
	time.Sleep(time.Duration(rand.Intn(10)) * time.Second)
	return nil
}
