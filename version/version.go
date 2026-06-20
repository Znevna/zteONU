package version

import (
	"fmt"
)

var (
	version = "0.0.8"
	appName = "ZteONU"
	date    = "20/06/2026"
	intro   = "https://github.com/Znevna/zteONU"
)

func Show() {
	fmt.Printf("%s %s, built at %s\nsource: %s\n", appName, version, date, intro)
}
