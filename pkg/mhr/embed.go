package mhr

import "embed"

//go:embed all:runtime
var runtimeFS embed.FS

// RuntimeFS returns the embedded filesystem containing mhr and mhr-cfw runtimes.
func RuntimeFS() embed.FS {
	return runtimeFS
}
