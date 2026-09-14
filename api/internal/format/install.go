package format

import "slices"

const (
	extensionKind     = "extension"
	installCapability = "extension-install"
)

// InstalledKinds names the kinds an app installs and runs, where every other kind is content it reads.
func InstalledKinds() []string {
	return []string{extensionKind}
}

// InstallCapabilities names what an instance must declare, one entry per app, before an asset of this kind is sent to it.
func InstallCapabilities(kind string, formats []string) []string {
	if !slices.Contains(InstalledKinds(), kind) {
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
