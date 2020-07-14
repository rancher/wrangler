package args

import (
	"fmt"
	"os"
	"testing"
)

func TestScan(t *testing.T) {
	cwd, _ := os.Getwd()
	fmt.Println(cwd)
	ScanDirectory("./testdata")
}
