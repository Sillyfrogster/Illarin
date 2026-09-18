package api_test

import (
	"testing"

	"github.com/Sillyfrogster/Illarin/api/internal/api"
)

type aliasedShape struct {
	Type       string `json:"type"`
	Visibility string `json:"visibility,omitempty"`
}

var shapeAliases = map[string]string{"type": "kind", "visibility": "discovery"}

func (s aliasedShape) MarshalJSON() ([]byte, error) {
	type plain aliasedShape
	return api.MarshalAliased(plain(s), shapeAliases)
}

func (s *aliasedShape) UnmarshalJSON(data []byte) error {
	type plain aliasedShape
	return api.UnmarshalAliased(data, (*plain)(s), shapeAliases)
}

func TestAResponseRepeatsEveryRenamedFieldUnderItsOldName(t *testing.T) {
	t.Parallel()

	encoded, err := aliasedShape{Type: "character"}.MarshalJSON()
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	if string(encoded) != `{"kind":"character","type":"character"}` {
		t.Fatalf("encoded = %s", encoded)
	}
}

func TestARequestTakesEitherTheNewNameOrTheOldOne(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		body string
		want string
	}{
		{"the new name", `{"type":"preset"}`, "preset"},
		{"the old name", `{"kind":"preset"}`, "preset"},
		{"both, the new one winning", `{"type":"preset","kind":"theme"}`, "preset"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var read aliasedShape
			if err := read.UnmarshalJSON([]byte(tc.body)); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			if read.Type != tc.want {
				t.Fatalf("type = %q, want %q", read.Type, tc.want)
			}
		})
	}
}

func TestARequestStillRefusesAFieldThatIsNeitherNameNorAlias(t *testing.T) {
	t.Parallel()

	var read aliasedShape
	if err := read.UnmarshalJSON([]byte(`{"type":"preset","shape":"round"}`)); err == nil {
		t.Fatal("an unknown field was accepted")
	}
}
