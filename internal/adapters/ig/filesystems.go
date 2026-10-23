package ig

import (
	"fmt"
	"strings"

	"golang.org/x/sys/unix"
)

// requiredFilesystems mirrors the check in IG's host.Init (pkg/utils/host,
// v0.57.0), which calls os.Exit instead of returning an error when one is
// missing. Checking first turns that into an error. Revisit on IG upgrades.
var requiredFilesystems = []struct {
	name  string
	paths []string // any one of them
	magic int64
}{
	{"debugfs", []string{"/sys/kernel/debug"}, unix.DEBUGFS_MAGIC},
	{"tracefs", []string{"/sys/kernel/tracing", "/sys/kernel/debug/tracing"}, unix.TRACEFS_MAGIC},
}

// statfsType returns the filesystem type (magic number) mounted at path.
func statfsType(path string) (int64, error) {
	var st unix.Statfs_t
	if err := unix.Statfs(path, &st); err != nil {
		return 0, err
	}
	return int64(st.Type), nil
}

// checkFilesystems returns an error naming the required filesystems that are
// not mounted.
func checkFilesystems(statfs func(path string) (int64, error)) error {
	var missing []string
next:
	for _, fs := range requiredFilesystems {
		for _, p := range fs.paths {
			if typ, err := statfs(p); err == nil && typ == fs.magic {
				continue next
			}
		}
		missing = append(missing, fs.name)
	}
	if len(missing) > 0 {
		return fmt.Errorf("filesystems not mounted: %s", strings.Join(missing, ", "))
	}
	return nil
}
