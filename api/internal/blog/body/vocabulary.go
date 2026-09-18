package body

var Languages = []string{
	"plain", "bash", "css", "diff", "go", "html", "javascript", "json",
	"markdown", "python", "rust", "sql", "toml", "typescript", "yaml",
}

var CalloutTypes = []string{"note", "tip", "important", "warning"}

func isLanguage(name string) bool {
	return contains(Languages, name)
}

func isCalloutType(calloutType string) bool {
	return contains(CalloutTypes, calloutType)
}

func contains(names []string, name string) bool {
	for _, one := range names {
		if one == name {
			return true
		}
	}
	return false
}
