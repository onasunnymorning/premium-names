package activities

import (
	"context"
	"errors"
	"os"
	"path/filepath"

	"github.com/yourorg/zone-names/internal/types"
)

// CleanupScratch removes the workflow's scratch subdirectory under the configured scratch root.
// It is safe to call even if the directory doesn't exist.
func (a *Activities) CleanupScratch(ctx context.Context, p types.CleanupParams) error {
	sub := filepath.Clean(p.ScratchSubdir)
	if sub == "." || sub == "" || sub == "/" || sub == ".." {
		// Safety: never allow deleting the entire scratch root or going up the tree.
		return errors.New("invalid scratch subdir for cleanup")
	}
	base := filepath.Join(a.cfg.ScratchDir, sub)
	// RemoveAll is idempotent; ignore if not exists
	if err := os.RemoveAll(base); err != nil {
		return err
	}
	return nil
}
