package modules

import (
	"testing"

	"github.com/Sillyfrogster/Illarin/api/internal/block"
	"github.com/Sillyfrogster/Illarin/api/internal/format"
	"github.com/google/uuid"
)

func TestACharacterWithExpressionsReachesEachAppInAFileItReadsThemFrom(t *testing.T) {
	t.Parallel()
	registry, err := Registry()
	if err != nil {
		t.Fatal(err)
	}
	offered := registry.OfferedFormats(format.CapabilitySubject{Type: "character", Elements: []block.Element{
		{ID: uuid.New(), Type: block.TypeTextSet, Role: block.RoleGreetings, Content: block.TextSet{
			Texts: []block.TextItem{{ID: uuid.New(), Text: "Hello"}},
		}},
		{ID: uuid.New(), Type: block.TypeImageSet, Role: block.RoleExpressions, Content: block.ImageSet{
			Images: []block.ImageItem{{ID: uuid.New(), MediaID: uuid.New(), Name: "joy"}},
		}},
	}})
	want := map[string]string{"sillytavern": "charx", "lumiverse": "charx"}
	for _, picked := range format.AppFormats(offered, registry) {
		if wanted, checked := want[picked.ID]; checked && picked.Format != wanted {
			t.Errorf("%s gets %s, want %s", picked.ID, picked.Format, wanted)
		}
	}
}
