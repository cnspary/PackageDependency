package models

import (
	"golang.org/x/mod/module"
)

// Module represents a Go module with its path and version.
type Module struct {
	Version module.Version
}