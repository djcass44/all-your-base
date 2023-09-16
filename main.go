package main

import (
	"fmt"
	"github.com/djcass44/all-your-base/cmd"
	"time"
)

var (
	version = "0.0.0"
	commit  = "develop"
	date    = time.Time{}.String()
)

func main() {
	cmd.Execute(fmt.Sprintf("%s-%s (%s)", version, commit, date))
}
