package work

import (
	"archive/zip"
	"context"
	"fmt"
	"testing"

	"github.com/Sillyfrogster/Illarin/api/internal/format"
)

func TestOnlySourceLocalImageReadErrorsDegrade(t *testing.T) {
	t.Parallel()
	if !localImageReadFailure(zip.ErrChecksum) {
		t.Error("a corrupt optional ZIP image did not degrade locally")
	}
	if localImageReadFailure(fmt.Errorf("read store: %w", format.ErrRangeRead)) {
		t.Error("an infrastructure range-read failure degraded as corrupt optional media")
	}
	if localImageReadFailure(context.Canceled) {
		t.Error("cancellation degraded as corrupt optional media")
	}
}
