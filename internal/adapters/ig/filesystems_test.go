package ig

import (
	"errors"
	"testing"

	"golang.org/x/sys/unix"
)

func fakeStatfs(mounts map[string]int64) func(string) (int64, error) {
	return func(path string) (int64, error) {
		if typ, ok := mounts[path]; ok {
			return typ, nil
		}
		return 0, errors.New("no such file")
	}
}

func TestCheckFilesystems(t *testing.T) {
	tests := []struct {
		name    string
		mounts  map[string]int64
		wantErr string
	}{
		{"all mounted", map[string]int64{
			"/sys/kernel/debug":   unix.DEBUGFS_MAGIC,
			"/sys/kernel/tracing": unix.TRACEFS_MAGIC,
		}, ""},
		{"tracefs under debugfs", map[string]int64{
			"/sys/kernel/debug":         unix.DEBUGFS_MAGIC,
			"/sys/kernel/debug/tracing": unix.TRACEFS_MAGIC,
		}, ""},
		{"nothing mounted", nil, "filesystems not mounted: debugfs, tracefs"},
		{"wrong type", map[string]int64{
			"/sys/kernel/debug":   unix.TMPFS_MAGIC,
			"/sys/kernel/tracing": unix.TRACEFS_MAGIC,
		}, "filesystems not mounted: debugfs"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := checkFilesystems(fakeStatfs(tt.mounts))
			if tt.wantErr == "" && err != nil {
				t.Errorf("unexpected error %v", err)
			}
			if tt.wantErr != "" && (err == nil || err.Error() != tt.wantErr) {
				t.Errorf("err = %v, want %q", err, tt.wantErr)
			}
		})
	}
}
