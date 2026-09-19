package format

// preservedLabels says what each namespace of preserved data holds, keyed by the name a file carried it under
var preservedLabels = map[string]string{
	"card":                       "unread character-card details",
	"character_book":             "extra lorebook details",
	"chub":                       "Chub metadata",
	"tavern_helper":              "TavernHelper scripts",
	"risuai":                     "RisuAI details",
	"lumiverse_modules":          "Lumiverse modules",
	"landing_perspective_layers": "perspective layers",
	"regex_scripts":              "regex scripts",
}

// PreservedLabel names a namespace of preserved data in words a reader recognises
func PreservedLabel(namespace string) string {
	if label, known := preservedLabels[namespace]; known {
		return label
	}
	return "other format-specific details"
}
