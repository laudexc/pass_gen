package main

import (
	"fmt"
	"os"

	"github.com/getkin/kin-openapi/openapi3"
)

func main() {
	specPath := "docs/openapi.yaml"
	if len(os.Args) > 1 {
		specPath = os.Args[1]
	}

	loader := &openapi3.Loader{IsExternalRefsAllowed: true}
	doc, err := loader.LoadFromFile(specPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "openapi load failed: %v\n", err)
		os.Exit(1)
	}

	if err := doc.Validate(loader.Context); err != nil {
		fmt.Fprintf(os.Stderr, "openapi validation failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("openapi valid: %s\n", specPath)
}
