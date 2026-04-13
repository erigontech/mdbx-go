//go:build ignore

package main

import (
	"fmt"
	"strings"

	"github.com/erigontech/mdbx-go/mdbx"
)

func main() {
	buildOpts := mdbx.BuildOptions()

	fmt.Println("=== MDBX Build Options ===")
	fmt.Println(buildOpts)
	fmt.Println()

	// Check for MDBX_USE_FALLOCATE specifically
	if strings.Contains(buildOpts, "MDBX_USE_FALLOCATE=1") || strings.Contains(buildOpts, "MDBX_USE_FALLOCATE=AUTO=1") {
		fmt.Println("✓ MDBX_USE_FALLOCATE is ENABLED (1)")
	} else if strings.Contains(buildOpts, "MDBX_USE_FALLOCATE=0") || strings.Contains(buildOpts, "MDBX_USE_FALLOCATE=AUTO=0") {
		fmt.Println("✗ MDBX_USE_FALLOCATE is DISABLED (0)")
	} else {
		fmt.Println("? MDBX_USE_FALLOCATE status unknown")
	}

	// Also check _GNU_SOURCE status
	if strings.Contains(buildOpts, "_GNU_SOURCE=YES") {
		fmt.Println("✓ _GNU_SOURCE is defined")
	} else if strings.Contains(buildOpts, "_GNU_SOURCE=NO") {
		fmt.Println("✗ _GNU_SOURCE is NOT defined")
	}
}
