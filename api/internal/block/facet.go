package block

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"slices"
)

type FacetKey string

const (
	FacetLorebook           FacetKey = "lorebook"
	FacetAlternateGreetings FacetKey = "alternate_greetings"
	FacetExpressions        FacetKey = "expressions"
	FacetGallery            FacetKey = "gallery"
	FacetVariables          FacetKey = "variables"
	FacetRegexScripts       FacetKey = "regex_scripts"
	FacetItems              FacetKey = "items"
)

const NoCeiling = -1

type Bucket struct {
	Value string
	Label string
	Min   int
	Max   int
}

func (b Bucket) Holds(count int) bool {
	if count < b.Min {
		return false
	}
	return b.Max == NoCeiling || count <= b.Max
}

type Facet struct {
	Key     FacetKey
	Label   string
	Role    Role
	Measure func(Content) int
	Buckets []Bucket
}

func (f Facet) Bucket(value string) (Bucket, bool) {
	for _, bucket := range f.Buckets {
		if bucket.Value == value {
			return bucket, true
		}
	}
	return Bucket{}, false
}

func presence() []Bucket {
	return []Bucket{
		{Value: "true", Label: "Included", Min: 1, Max: NoCeiling},
		{Value: "false", Label: "None", Min: 0, Max: 0},
	}
}

func countEntries(content Content) int {
	table, ok := content.(EntryTable)
	if !ok {
		return 0
	}
	return len(table.Entries)
}

func countAlternateGreetings(content Content) int {
	set, ok := content.(TextSet)
	if !ok {
		return 0
	}
	written := 0
	for _, item := range set.Texts {
		if item.Text != "" {
			written++
		}
	}
	return max(0, written-1)
}

func countImages(content Content) int {
	set, ok := content.(ImageSet)
	if !ok {
		return 0
	}
	return len(set.Images)
}

func countVariables(content Content) int {
	schema, ok := content.(VariableSchema)
	if !ok {
		return 0
	}
	return len(schema.Variables)
}

func countScripts(content Content) int {
	list, ok := content.(ScriptList)
	if !ok {
		return 0
	}
	return len(list.Scripts)
}

func countRecords(content Content) int {
	list, ok := content.(RecordList)
	if !ok {
		return 0
	}
	return len(list.Records) + len(list.LoomItems)
}

var characterFacets = []Facet{
	{
		Key: FacetLorebook, Label: "Lorebook", Role: RoleLorebookEntries,
		Measure: countEntries, Buckets: presence(),
	},
	{
		Key: FacetAlternateGreetings, Label: "Alternate greetings", Role: RoleGreetings,
		Measure: countAlternateGreetings,
		Buckets: []Bucket{
			{Value: "1", Label: "1", Min: 1, Max: 1},
			{Value: "2-4", Label: "2 to 4", Min: 2, Max: 4},
			{Value: "5-up", Label: "5 or more", Min: 5, Max: NoCeiling},
		},
	},
	{
		Key: FacetExpressions, Label: "Expressions", Role: RoleExpressions,
		Measure: countImages, Buckets: presence(),
	},
	{
		Key: FacetGallery, Label: "Gallery", Role: RoleGallery,
		Measure: countImages, Buckets: presence(),
	},
}

var presetFacets = []Facet{
	{
		Key: FacetVariables, Label: "Variables", Role: RolePromptVariables,
		Measure: countVariables, Buckets: presence(),
	},
	{
		Key: FacetRegexScripts, Label: "Regex scripts", Role: RoleRegexScripts,
		Measure: countScripts, Buckets: presence(),
	},
}

var packFacets = []Facet{
	{
		Key: FacetItems, Label: "Items", Role: RolePackItems,
		Measure: countRecords,
		Buckets: []Bucket{
			{Value: "1", Label: "1 item", Min: 1, Max: 1},
			{Value: "2-9", Label: "2 to 9", Min: 2, Max: 9},
			{Value: "10-up", Label: "10 or more", Min: 10, Max: NoCeiling},
		},
	},
}

var facets = map[string][]Facet{
	"character": characterFacets,
	"preset":    presetFacets,
	"pack":      packFacets,
}

func Facets(workType string) []Facet { return facets[workType] }

func FacetByKey(workType string, key string) (Facet, bool) {
	for _, facet := range Facets(workType) {
		if string(facet.Key) == key {
			return facet, true
		}
	}
	return Facet{}, false
}

func MeasureFacets(workType string, elements []Element) map[FacetKey]int {
	facets := Facets(workType)
	counts := make(map[FacetKey]int, len(facets))
	for _, facet := range facets {
		total := 0
		for _, element := range elements {
			if element.Role != facet.Role || element.Content == nil {
				continue
			}
			total += facet.Measure(element.Content)
		}
		counts[facet.Key] = total
	}
	return counts
}

func FacetStamp() string {
	types := make([]string, 0, len(facets))
	for workType := range facets {
		types = append(types, workType)
	}
	slices.Sort(types)

	digest := sha256.New()
	for _, workType := range types {
		for _, facet := range facets[workType] {
			fmt.Fprintf(digest, "facet\x00%s\x00%s\x00%s\x00%s\n",
				workType, facet.Key, facet.Label, facet.Role)
			for _, bucket := range facet.Buckets {
				fmt.Fprintf(digest, "bucket\x00%s\x00%d\x00%d\n",
					bucket.Value, bucket.Min, bucket.Max)
			}
		}
	}
	return hex.EncodeToString(digest.Sum(nil))[:16]
}
