package upload

import (
	"testing"

	"github.com/gin-gonic/gin"
)

func TestAStoredReplacementPreviewReturnsArraysInsteadOfNull(t *testing.T) {
	t.Parallel()
	response := toAPIIngest(Operation{
		Status:  IngestPreview,
		Preview: &Preview{},
	})
	preview := response["preview"].(gin.H)
	for _, field := range []string{"conflicts", "unrepresentable", "missingWording"} {
		values, ok := preview[field].([]string)
		if !ok || values == nil || len(values) != 0 {
			t.Fatalf("%s = %#v, want an empty array", field, preview[field])
		}
	}
}
