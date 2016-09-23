package args

import (
	"flag"
)

func init() {
	// Windows specific options.
	flag.StringVar(&args.VCVersion, "vc_version", "14.0",
		"(Windows Only) The Visual Studio version to use.")
}
