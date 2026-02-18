package creds

import (
	"fmt"
	"os"
)

// runtimeDirFallback returns the default runtime directory on Linux.
// systemd/pam typically provides /run/user/<uid> as a tmpfs mount.
func runtimeDirFallback() string {
	return fmt.Sprintf("/run/user/%d", os.Getuid())
}
