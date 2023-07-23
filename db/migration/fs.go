// This file is intended only to export FS variable
// due to the current limitation of go:embed directive,
// which does not support paths with '..' or '.'.
// See: https://github.com/golang/go/issues/46056
package migration

import (
	"embed"
)

//go:embed scripts/*.sql
var FS embed.FS
