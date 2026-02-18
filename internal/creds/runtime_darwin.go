package creds

import (
	"fmt"
	"os"
	"path/filepath"
)

// runtimeDirFallback returns the default runtime directory on macOS.
// macOS has no /run filesystem; we use a per-user directory under
// the system temp directory instead.
func runtimeDirFallback() string {
	return filepath.Join(os.TempDir(), fmt.Sprintf("vop-%d", os.Getuid()))
}
