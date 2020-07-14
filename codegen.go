//go:generate /bin/rm -rf pkg/generated
//go:generate go run pkg/codegen/main.go

package main

import (
	"fmt"
)

var (
	Version   = "v0.0.0-dev"
	GitCommit = "v0.0.0-dev"
)

func main() {
	fmt.Println("Run go generate")
}
