package upload

import (
	"reflect"
	"strings"
	"testing"
)

func archive(entries ...string) findEntry {
	return inArchive("", "", func(entry string) bool {
		for _, held := range entries {
			if held == entry {
				return true
			}
		}
		return false
	})
}

func titles(page readmePage) []string {
	found := []string{}
	for _, section := range page.Sections {
		found = append(found, section.Title)
	}
	return found
}

func TestSectionsFollowTheShallowestHeadingUnderTheTitle(t *testing.T) {
	t.Parallel()
	page := readReadme(strings.Join([]string{
		"# Quiet Toolbox",
		"",
		"Small tools for a calmer chat.",
		"",
		"## Install",
		"",
		"Clone it.",
		"",
		"### From a zip",
		"",
		"Unpack it.",
		"",
		"## Usage",
		"Type a command.",
	}, "\n"), archive())

	if page.Opening.Text != "Small tools for a calmer chat." {
		t.Errorf("opening = %q, want the text before the first section", page.Opening.Text)
	}
	if got := titles(page); !reflect.DeepEqual(got, []string{"Install", "Usage"}) {
		t.Fatalf("sections = %v, want Install and Usage", got)
	}
	if want := "Clone it.\n\n### From a zip\n\nUnpack it."; page.Sections[0].Text != want {
		t.Errorf("install = %q, want the deeper heading kept inside it: %q", page.Sections[0].Text, want)
	}
	if page.Sections[1].Text != "Type a command." {
		t.Errorf("usage = %q", page.Sections[1].Text)
	}
}

func TestSeveralTopHeadingsAreAllSections(t *testing.T) {
	t.Parallel()
	page := readReadme("# Setup\n\nOne.\n\n# Usage\n\nTwo.", archive())
	if got := titles(page); !reflect.DeepEqual(got, []string{"Setup", "Usage"}) {
		t.Fatalf("sections = %v, want both level-one headings", got)
	}
	if page.Opening.Text != "" {
		t.Errorf("opening = %q, want nothing", page.Opening.Text)
	}
}

func TestAReadmeWithoutSectionsIsAllOpening(t *testing.T) {
	t.Parallel()
	page := readReadme("# Dice\n\nRolls dice.\nTwo lines.", archive())
	if len(page.Sections) != 0 || page.Opening.Text != "Rolls dice.\nTwo lines." {
		t.Fatalf("page = %+v, want one opening", page)
	}
}

func TestANearlyEmptyReadmeSeedsNothing(t *testing.T) {
	t.Parallel()
	for _, source := range []string{"", "   \n", "# Randomizer\n", "# Randomizer\n\n## Notes\n"} {
		page := readReadme(source, archive())
		if page.Opening.Text != "" || len(page.Opening.Images) != 0 || len(page.Sections) != 0 || page.Cover != nil {
			t.Errorf("%q seeded %+v, want nothing", source, page)
		}
	}
}

func TestCodeAndTablesStayAsWritten(t *testing.T) {
	t.Parallel()
	fence := "```\n## not a heading\n/roll 2d6\n```"
	table := "| Command | Does |\n| --- | --- |\n| `/roll` | Rolls |"
	page := readReadme("# Dice\n\n## Commands\n\n"+fence+"\n\n"+table+"\n\n~~~js\nroll()\n~~~", archive())
	if got := titles(page); !reflect.DeepEqual(got, []string{"Commands"}) {
		t.Fatalf("sections = %v, want the fence's heading left inside the fence", got)
	}
	if want := fence + "\n\n" + table + "\n\n~~~js\nroll()\n~~~"; page.Sections[0].Text != want {
		t.Errorf("commands = %q, want %q", page.Sections[0].Text, want)
	}
}

func TestRulesBetweenSectionsAreLeftOutAndUnderlinedHeadingsKept(t *testing.T) {
	t.Parallel()
	page := readReadme("# Setup\n\nIntro.\n\n---\n\n# Usage\n\nSettings\n--------\n\nOpen them.\n\n***\n", archive())
	if got := titles(page); !reflect.DeepEqual(got, []string{"Setup", "Usage"}) {
		t.Fatalf("sections = %v", got)
	}
	if page.Sections[0].Text != "Intro." {
		t.Errorf("setup = %q, want the rule left out", page.Sections[0].Text)
	}
	if want := "Settings\n--------\n\nOpen them."; page.Sections[1].Text != want {
		t.Errorf("usage = %q, want %q", page.Sections[1].Text, want)
	}
}

func TestImagesInTheArchiveMoveBesideTheirSection(t *testing.T) {
	t.Parallel()
	page := readReadme(strings.Join([]string{
		"# Model",
		"",
		"## Settings",
		"",
		"Pick a model.",
		"",
		"![Global settings](readme_img/ui%20global.png)",
		"",
		"![Missing](readme_img/gone.png)",
		"![Remote](https://example.com/shot.png)",
		"",
		`<p align="center"><a href="docs/list.png"><img src="./docs/list.png" alt="Model list" width="360"></a></p>`,
		"",
		"[![Global again](readme_img/ui%20global.png)](readme_img/ui%20global.png)",
		"",
		"Then save.",
	}, "\n"), archive("readme_img/ui global.png", "docs/list.png"))

	if len(page.Sections) != 1 {
		t.Fatalf("sections = %+v", page.Sections)
	}
	settings := page.Sections[0]
	if settings.Text != "Pick a model.\n\nThen save." {
		t.Errorf("text = %q, want every picture taken out of it", settings.Text)
	}
	want := []readmeImage{{Entry: "readme_img/ui global.png", Name: "Global settings"}, {Entry: "docs/list.png", Name: "Model list"}}
	if !reflect.DeepEqual(settings.Images, want) {
		t.Errorf("images = %+v, want %+v", settings.Images, want)
	}
}

func TestHTMLWithWordsStaysAndBareMarkupGoes(t *testing.T) {
	t.Parallel()
	page := readReadme(strings.Join([]string{
		`<div align="center">`,
		"",
		"# Mind",
		"",
		"<details><summary>More</summary>Hidden notes.</details>",
		"",
		"</div>",
	}, "\n"), archive())
	if want := "<details><summary>More</summary>Hidden notes.</details>"; page.Opening.Text != want {
		t.Errorf("opening = %q, want %q", page.Opening.Text, want)
	}
}

func TestTheFirstPictureBeforeAnySectionIsTheCover(t *testing.T) {
	t.Parallel()
	page := readReadme(strings.Join([]string{
		`<img src="https://example.com/badge.svg" alt="version">`,
		"",
		"# Suite",
		"",
		"![Banner](art/banner.png)",
		"",
		"![Second](art/second.png)",
		"",
		"A suite of tools.",
		"",
		"## Library",
		"",
		"![Library](art/library.png)",
	}, "\n"), archive("art/banner.png", "art/second.png", "art/library.png"))

	if page.Cover == nil || *page.Cover != (readmeImage{Entry: "art/banner.png", Name: "Banner"}) {
		t.Fatalf("cover = %+v, want the banner", page.Cover)
	}
	if want := []readmeImage{{Entry: "art/second.png", Name: "Second"}}; !reflect.DeepEqual(page.Opening.Images, want) {
		t.Errorf("opening images = %+v, want %+v", page.Opening.Images, want)
	}
	if page.Opening.Text != "A suite of tools." {
		t.Errorf("opening = %q", page.Opening.Text)
	}
}

func TestAPictureOnlyInASectionIsNeverTheCover(t *testing.T) {
	t.Parallel()
	page := readReadme("# Suite\n\nIntro.\n\n## Library\n\n![Library](art/library.png)", archive("art/library.png"))
	if page.Cover != nil {
		t.Errorf("cover = %+v, want none", page.Cover)
	}
	if len(page.Sections) != 1 || page.Sections[0].Text != "" || len(page.Sections[0].Images) != 1 {
		t.Errorf("sections = %+v, want one section holding only its picture", page.Sections)
	}
}

func TestAContentsListOfLinksWithinTheReadmeIsLeftOut(t *testing.T) {
	t.Parallel()
	page := readReadme(strings.Join([]string{
		"# Mind",
		"",
		"## Table of contents",
		"",
		"1. [At a glance](#at-a-glance)",
		"2. [Install](#install)",
		"   - [From a zip](#from-a-zip)",
		"",
		"## Install",
		"",
		"See [the notes](#notes) first.",
	}, "\n"), archive())
	if got := titles(page); !reflect.DeepEqual(got, []string{"Install"}) {
		t.Fatalf("sections = %v, want the contents list left out", got)
	}
}

func TestHeadingsBecomePlainTitles(t *testing.T) {
	t.Parallel()
	page := readReadme(strings.Join([]string{
		"# T",
		"## The `/roll` and [link](https://example.com) *em* <b>bold</b> \\* &amp; more",
		"Text.",
		"## <script>alert(1)</script><style>h2{color:red}</style>Usage",
		"Text.",
	}, "\n"), archive())
	want := []string{"The /roll and link em bold * & more", "Usage"}
	if got := titles(page); !reflect.DeepEqual(got, want) {
		t.Fatalf("titles = %q, want %q", got, want)
	}
}

func TestAnAddressIsFoundBesideAReadmeInAFolderOfItsOwn(t *testing.T) {
	t.Parallel()
	find := inArchive("tool-main/", ".github/", func(entry string) bool {
		return entry == "tool-main/.github/shot.png" || entry == "tool-main/art/logo.png"
	})
	cases := map[string]string{
		"shot.png":         "tool-main/.github/shot.png",
		"../art/logo.png":  "tool-main/art/logo.png",
		"/art/logo.png":    "tool-main/art/logo.png",
		"../../escape.png": "",
	}
	for address, wanted := range cases {
		if entry, found := find(address); found != (wanted != "") || found && entry != wanted {
			t.Errorf("find(%q) = %q, %t, want %q", address, entry, found, wanted)
		}
	}
}

func TestAnAddressIsFoundOnlyInsideTheArchiveFolder(t *testing.T) {
	t.Parallel()
	find := inArchive("tool-main/", "", func(entry string) bool {
		return entry == "tool-main/docs/a b.png" || entry == "outside.png"
	})
	cases := map[string]bool{
		"docs/a%20b.png":                 true,
		"./docs/a b.png":                 true,
		"/docs/a%20b.png":                true,
		"docs/a%20b.png?raw=true":        true,
		"docs/a%20b.png#top":             true,
		"../outside.png":                 false,
		"https://example.com/docs/a.png": false,
		"//example.com/docs/a.png":       false,
		"data:image/png;base64,AAAA":     false,
		"":                               false,
	}
	for address, wanted := range cases {
		if _, found := find(address); found != wanted {
			t.Errorf("find(%q) = %t, want %t", address, found, wanted)
		}
	}
}

func TestATableOfPicturesMovesIntoTheSectionPictures(t *testing.T) {
	t.Parallel()
	page := readReadme(strings.Join([]string{
		"# Tool",
		"",
		"## Screenshots",
		"",
		"| Home | Editor |",
		"| ---- | ------ |",
		"| ![Home](shots/home.png) | ![](shots/editor.png) |",
		"",
		"| Setting | Default |",
		"| ------- | ------- |",
		"| Speed | ![Fast](shots/fast.png) |",
	}, "\n"), archive("shots/home.png", "shots/editor.png", "shots/fast.png"))

	section := page.Sections[0]
	var names []string
	for _, image := range section.Images {
		names = append(names, image.Name)
	}
	if strings.Join(names, ",") != "Home,Editor,Fast" {
		t.Errorf("pictures = %q, want every picture the tables hold, the unnamed one named by its column", names)
	}
	if strings.Contains(section.Text, "home.png") || !strings.Contains(section.Text, "| Speed | ![Fast](shots/fast.png) |") {
		t.Errorf("text = %q, want the table of only pictures gone and the table with words kept", section.Text)
	}
}

func TestARemotePictureIsListedByItsAddress(t *testing.T) {
	t.Parallel()
	page := readReadme(strings.Join([]string{
		"# Tool",
		"",
		"![Build](https://img.shields.io/badge/build-passing)",
		"![Logo](https://example.com/logo.svg)",
		"![Hero](https://cdn.example.com/hero.jpg?raw=1)",
		"",
		"## Screens",
		"",
		"![Chat](https://example.com/chat.png)",
		"",
		"![Gone](art/gone.png)",
	}, "\n"), archive())

	if got := page.Opening.Remote; len(got) != 1 || got[0] != (remoteImage{Address: "https://cdn.example.com/hero.jpg?raw=1", Name: "Hero"}) {
		t.Errorf("opening remote = %+v, want the hero and neither the badge nor the vector drawing", got)
	}
	if len(page.Sections) != 1 {
		t.Fatalf("sections = %q, want the section that holds only a remote picture", titles(page))
	}
	if got := page.Sections[0].Remote; len(got) != 1 || got[0].Address != "https://example.com/chat.png" || got[0].Name != "Chat" {
		t.Errorf("section remote = %+v, want the chat picture and not the missing archived one", got)
	}
}

func TestAnAnchorWithNoWordsIsLeftOut(t *testing.T) {
	t.Parallel()
	page := readReadme("<a name=\"top\"></a>\n\nHello there.\n\n<a name=\"top\">Jump</a>\n", archive())
	if page.Opening.Text != "Hello there.\n\n<a name=\"top\">Jump</a>" {
		t.Errorf("opening = %q, want the bare anchor gone and the worded one kept", page.Opening.Text)
	}
}

func TestAPasteStartsAPieceAtEveryHeadingBelowTheTitle(t *testing.T) {
	t.Parallel()
	page := readPaste(strings.Join([]string{
		"[Back](https://rentry.org/harbor-index)",
		"# **->%teal% Harbor Notes%%->**",
		"",
		"!!! info An unofficial guide.",
		"",
		"### Lantern Prompt",
		"",
		"Opening words.",
		"",
		"## V2",
		"",
		"#### System Prompt",
		"```",
		"Stay in the lighthouse.",
		"```",
		"#### Post History",
		"```",
		"Avoid repetition.",
		"```",
		"",
		"[Back](https://rentry.org/harbor-index)",
	}, "\n"))

	if page.Title != "Harbor Notes" {
		t.Errorf("title = %q, want the heading's words without rentry's marks", page.Title)
	}
	if page.Opening.Text != "!!! info An unofficial guide." {
		t.Errorf("opening = %q, want the text under the title and no navigation link", page.Opening.Text)
	}
	if got := titles(page); !reflect.DeepEqual(got, []string{"Lantern Prompt", "V2 · System Prompt", "Post History"}) {
		t.Fatalf("sections = %v, want one per heading with the empty V2 folded into the next", got)
	}
	if want := "```\nAvoid repetition.\n```"; page.Sections[2].Text != want {
		t.Errorf("last section = %q, want %q without the closing navigation link", page.Sections[2].Text, want)
	}
}

func TestAPasteWithoutHeadingsIsAllOpening(t *testing.T) {
	t.Parallel()
	page := readPaste("[Back](https://rentry.org/elsewhere)\n\nJust a note.")
	if page.Title != "" || len(page.Sections) != 0 || page.Opening.Text != "[Back](https://rentry.org/elsewhere)\n\nJust a note." {
		t.Errorf("page = %+v, want every word in the opening when nothing is a title", page)
	}
}
