package common

import (
	"os"
)

var (
	CurrentDir string
)

func init() {
	CurrentDir, _ = os.Getwd()
}
