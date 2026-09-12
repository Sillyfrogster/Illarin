package http

import (
	"testing"

	"github.com/Sillyfrogster/Illarin/api/internal/asset"
	"github.com/gin-gonic/gin"
)

func TestAStoredReplacementPreviewReturnsArraysInsteadOfNull(t *testing.T) {
	response := toAPIIngest(asset.IngestOperation{
		Status:  asset.IngestPreview,
		Preview: &asset.ReplacementPreview{},
	})
	preview := response["preview"].(gin.H)
	for _, field := range []string{"conflicts", "unrepresentable", "missingWording"} {
		values, ok := preview[field].([]string)
		if !ok || values == nil || len(values) != 0 {
			t.Fatalf("%s = %#v, want an empty array", field, preview[field])
		}
	}
}
