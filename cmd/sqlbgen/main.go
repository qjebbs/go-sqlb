package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/qjebbs/go-sqlb/internal/generate"
)

func main() {
	var unifile bool
	flag.BoolVar(&unifile, "unifile", false, "generate a single file for the entire package")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [flags] [patterns]\n", os.Args[0])
		fmt.Fprintln(os.Stderr, "Flags:")
		flag.PrintDefaults()
		fmt.Fprintln(os.Stderr, "Patterns:")
		fmt.Fprintln(os.Stderr, "  (default ./...)")
	}
	flag.Parse()

	patterns := flag.Args()
	if len(patterns) == 0 {
		patterns = []string{"./..."}
	}

	err := generate.NewGenerator(unifile).Generate(patterns)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
