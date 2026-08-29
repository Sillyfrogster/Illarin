package postdoc

// Languages is every label a code block may carry. The site derives its
// highlighting from this label and the code beside it.
var Languages = []string{
	"plain", "bash", "css", "diff", "go", "html", "javascript", "json",
	"markdown", "python", "rust", "sql", "toml", "typescript", "yaml",
}

// CalloutKinds is every kind of aside a callout may be.
var CalloutKinds = []string{"note", "tip", "important", "warning"}

func isLanguage(name string) bool {
	return contains(Languages, name)
}

func isCalloutKind(kind string) bool {
	return contains(CalloutKinds, kind)
}

func contains(names []string, name string) bool {
	for _, one := range names {
		if one == name {
			return true
		}
	}
	return false
}
