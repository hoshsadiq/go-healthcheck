package checkers

import (
	"context"
	"fmt"
	"os"

	"golang.org/x/sys/unix"

	"github.com/hoshsadiq/go-healthcheck"
)

type diskSpace struct {
	dir       string
	threshold uint64
	statfs    func(string, *unix.Statfs_t) error
}

// Check test if the filesystem disk usage is above threshold.
func (ds *diskSpace) Check(ctx context.Context) error {
	if _, err := os.Stat(ds.dir); err != nil {
		return fmt.Errorf("filesystem not found: %w", err)
	}

	fs := unix.Statfs_t{}
	err := ds.statfs(ds.dir, &fs)
	if err != nil {
		return fmt.Errorf("error looking for %s filesystem stats: %w", ds.dir, err)
	}

	total := fs.Blocks * uint64(fs.Bsize)
	free := fs.Bfree * uint64(fs.Bsize)
	used := total - free
	usedPercentage := 100 * used / total //nolint:gomnd
	if usedPercentage > ds.threshold {
		return fmt.Errorf("used: %d%% threshold: %d%% location: %s", usedPercentage, ds.threshold, ds.dir)
	}

	return nil
}

// DiskSpace returns a diskSpace health checker, which checks if filesystem usage is above the threshold which is defined in percentage.
func DiskSpace(dir string, threshold uint64) healthcheck.Checker {
	return &diskSpace{
		dir:       dir,
		threshold: threshold,
		statfs:    unix.Statfs,
	}
}
