package format

import "slices"

const (
	extensionType     = "extension"
	installCapability = "extension-install"
)

// InstalledTypes names the types an app installs and runs, where every other type is content it reads.
func InstalledTypes() []string {
	return []string{extensionType}
}

// InstallCapabilities names what an instance must declare, one entry per app, before a work of this type is sent to it.
func InstallCapabilities(workType string, formats []string) []string {
	if !slices.Contains(InstalledTypes(), workType) {
		return nil
	}
	needed := make([]string, 0, len(Apps()))
	for _, app := range Apps() {
		for _, format := range formats {
			if slices.Contains(app.Reads, format) {
				needed = append(needed, app.Namespace+":"+installCapability)
				break
			}
		}
	}
	return needed
}
