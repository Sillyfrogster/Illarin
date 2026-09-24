package upload

import (
	"html"
	"net/url"
	"path"
	"regexp"
	"slices"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	east "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
	markup "golang.org/x/net/html"
)

type readmeImage struct {
	Entry string
	Name  string
}

type remoteImage struct {
	Address string
	Name    string
}

type readmeSection struct {
	Title  string
	Text   string
	Images []readmeImage
	Remote []remoteImage
}

type readmePage struct {
	Title    string
	Cover    *readmeImage
	Opening  readmeSection
	Sections []readmeSection
}

type findEntry func(address string) (string, bool)

var parser = goldmark.New(goldmark.WithExtensions(extension.Table)).Parser()

// readReadme splits a README into sections at its shallowest heading below the title
func readReadme(source string, find findEntry) readmePage {
	return readMarkdown(source, find, false)
}

// readPaste splits pasted Markdown at every heading below its title, leaving out the page's navigation links
func readPaste(source string) readmePage {
	return readMarkdown(source, func(string) (string, bool) { return "", false }, true)
}

func readMarkdown(source string, find findEntry, everyHeading bool) readmePage {
	source = strings.TrimPrefix(strings.ReplaceAll(source, "\r\n", "\n"), "\uFEFF")
	raw := []byte(source)
	parts := divide(parser.Parse(text.NewReader(raw)), raw, find)
	skipped := map[int]bool{}
	if everyHeading {
		skipped = navigation(parts)
	}
	title := titlePart(parts, skipped)
	if title < 0 {
		skipped = map[int]bool{}
	}
	depth := sectionDepth(parts, title)
	starts := func(index int) bool {
		return parts[index].depth > 0 && (everyHeading || parts[index].depth == depth)
	}

	page := readmePage{}
	if title >= 0 {
		page.Title = parts[title].words
	}
	current := &page.Opening
	for index, part := range parts {
		switch {
		case index == title || skipped[index]:
		case starts(index):
			page.Sections = append(page.Sections, readmeSection{Title: part.words})
			current = &page.Sections[len(page.Sections)-1]
		default:
			current.add(part)
		}
	}
	page.Sections = kept(page.Sections, parts, title, starts, skipped, everyHeading)
	if len(page.Opening.Images) > 0 {
		cover := page.Opening.Images[0]
		page.Cover = &cover
		page.Opening.Images = page.Opening.Images[1:]
	}
	return page
}

// inArchive resolves a picture address against the README folder and never leaves the archive root
func inArchive(root, folder string, has func(entry string) bool) findEntry {
	return func(address string) (string, bool) {
		parsed, err := url.Parse(strings.ReplaceAll(strings.TrimSpace(address), " ", "%20"))
		if err != nil || parsed.Scheme != "" || parsed.Host != "" || parsed.Opaque != "" || parsed.Path == "" {
			return "", false
		}
		name, rooted := strings.CutPrefix(parsed.Path, "/")
		if !rooted {
			name = folder + name
		}
		name = path.Clean(name)
		if name == "." || name == ".." || strings.HasPrefix(name, "../") {
			return "", false
		}
		return root + name, has(root + name)
	}
}

type part struct {
	node     ast.Node
	depth    int
	words    string
	text     string
	images   []readmeImage
	remote   []remoteImage
	pictures bool
	anchors  bool
	links    bool
}

func (s *readmeSection) add(p part) {
	for _, image := range p.images {
		if !containsEntry(s.Images, image.Entry) {
			s.Images = append(s.Images, image)
		}
	}
	for _, picture := range p.remote {
		if !slices.Contains(s.Remote, picture) {
			s.Remote = append(s.Remote, picture)
		}
	}
	if p.pictures || p.text == "" {
		return
	}
	if s.Text != "" {
		s.Text += "\n\n"
	}
	s.Text += p.text
}

func divide(document ast.Node, source []byte, find findEntry) []part {
	var parts []part
	var starts []int
	for node := document.FirstChild(); node != nil; node = node.NextSibling() {
		if offset, ok := firstOffset(node, source); ok {
			parts = append(parts, part{node: node})
			starts = append(starts, lineStart(source, offset))
		}
	}
	for index := range parts {
		end := len(source)
		if index+1 < len(starts) {
			end = starts[index+1]
		}
		parts[index].text = tidy(string(source[starts[index]:end]))
		parts[index].read(source, find)
	}
	return parts
}

func (p *part) read(source []byte, find findEntry) {
	switch node := p.node.(type) {
	case *ast.Heading:
		p.depth = node.Level
		p.words = withoutRentryMarks(plain(node, source))
	case *ast.Paragraph:
		found, only := paragraphPictures(node, source)
		p.images, p.remote = located(found, find)
		p.pictures = only
		p.links = onlyLinks(node, source)
	case *ast.HTMLBlock:
		found, words := readHTML(p.text)
		p.images, p.remote = located(found, find)
		p.pictures = blank(words)
	case *east.Table:
		found, only := tablePictures(node, source)
		p.images, p.remote = located(found, find)
		p.pictures = only
	case *ast.List:
		p.anchors = onlyAnchors(node, source)
	}
}

// navigation finds link-only lines before the first heading, and the same line again at the very end
func navigation(parts []part) map[int]bool {
	skipped := map[int]bool{}
	seen := map[string]bool{}
	for index, candidate := range parts {
		if candidate.depth > 0 || !candidate.links {
			break
		}
		skipped[index] = true
		seen[candidate.text] = true
	}
	if last := len(parts) - 1; last > 0 && parts[last].links && seen[parts[last].text] {
		skipped[last] = true
	}
	return skipped
}

func titlePart(parts []part, skipped map[int]bool) int {
	for index, candidate := range parts {
		if candidate.pictures || skipped[index] {
			continue
		}
		if candidate.depth == 0 {
			return -1
		}
		for other, rest := range parts {
			if other != index && rest.depth > 0 && rest.depth <= candidate.depth {
				return -1
			}
		}
		return index
	}
	return -1
}

func sectionDepth(parts []part, title int) int {
	depth := 0
	for index, candidate := range parts {
		if index != title && candidate.depth > 0 && (depth == 0 || candidate.depth < depth) {
			depth = candidate.depth
		}
	}
	return depth
}

// kept leaves out sections holding nothing or only a contents list, and folds an empty heading into the next one's name when asked
func kept(
	sections []readmeSection, parts []part, title int, starts func(int) bool, skipped map[int]bool, fold bool,
) []readmeSection {
	anchorsOnly := make([]bool, 0, len(sections))
	for index, candidate := range parts {
		switch {
		case index == title || skipped[index]:
		case starts(index):
			anchorsOnly = append(anchorsOnly, true)
		case len(anchorsOnly) > 0 && !candidate.pictures:
			last := len(anchorsOnly) - 1
			anchorsOnly[last] = anchorsOnly[last] && candidate.anchors
		}
	}
	result := make([]readmeSection, 0, len(sections))
	carried := ""
	for index, section := range sections {
		pictured := len(section.Images) > 0 || len(section.Remote) > 0
		if carried != "" {
			section.Title = carried + " · " + section.Title
			carried = ""
		}
		if (section.Text != "" || pictured) && !(anchorsOnly[index] && !pictured) {
			result = append(result, section)
		} else if fold && section.Text == "" && !pictured {
			carried = section.Title
		}
	}
	return result
}

type picture struct {
	address string
	name    string
}

func located(found []picture, find findEntry) ([]readmeImage, []remoteImage) {
	var images []readmeImage
	var remote []remoteImage
	for _, candidate := range found {
		if entry, ok := find(candidate.address); ok {
			if !containsEntry(images, entry) {
				images = append(images, readmeImage{Entry: entry, Name: candidate.name})
			}
			continue
		}
		if shown, ok := remotePicture(candidate); ok && !slices.Contains(remote, shown) {
			remote = append(remote, shown)
		}
	}
	return images, remote
}

// badgeHosts serve status badges, which are not pictures of the extension
var badgeHosts = map[string]bool{"img.shields.io": true, "badgen.net": true, "badge.fury.io": true}

// remotePicture keeps a picture from another site when Illarin could hold a copy of it
func remotePicture(candidate picture) (remoteImage, bool) {
	address := strings.TrimSpace(candidate.address)
	parsed, err := url.Parse(address)
	if err != nil || (parsed.Scheme != "https" && parsed.Scheme != "http") || parsed.Host == "" {
		return remoteImage{}, false
	}
	if badgeHosts[parsed.Host] || strings.EqualFold(path.Ext(parsed.Path), ".svg") {
		return remoteImage{}, false
	}
	return remoteImage{Address: address, Name: candidate.name}, true
}

func paragraphPictures(paragraph ast.Node, source []byte) ([]picture, bool) {
	var found []picture
	only, markup := true, false
	for child := paragraph.FirstChild(); child != nil; child = child.NextSibling() {
		switch node := child.(type) {
		case *ast.Image:
			found = append(found, picture{address: string(node.Destination), name: plain(node, source)})
		case *ast.Link:
			inside, pictures := paragraphPictures(node, source)
			found = append(found, inside...)
			only = only && pictures && len(inside) > 0
		case *ast.RawHTML:
			inside, words := readHTML(string(node.Segments.Value(source)))
			found = append(found, inside...)
			only, markup = only && blank(words), true
		case *ast.Text:
			only = only && blank(string(node.Segment.Value(source)))
		default:
			only = false
		}
	}
	return found, only && (len(found) > 0 || markup)
}

func tablePictures(table *east.Table, source []byte) ([]picture, bool) {
	var found []picture
	var captions []string
	only := true
	for row := table.FirstChild(); row != nil; row = row.NextSibling() {
		_, header := row.(*east.TableHeader)
		column := 0
		for cell := row.FirstChild(); cell != nil; cell = cell.NextSibling() {
			inside, pictures := paragraphPictures(cell, source)
			if header {
				captions = append(captions, plain(cell, source))
			} else if cell.HasChildren() {
				only = only && pictures
			}
			for _, candidate := range inside {
				if blank(candidate.name) && column < len(captions) {
					candidate.name = captions[column]
				}
				found = append(found, candidate)
			}
			column++
		}
	}
	return found, only && len(found) > 0
}

func readHTML(fragment string) ([]picture, string) {
	var found []picture
	var words strings.Builder
	hidden := false
	tokens := markup.NewTokenizer(strings.NewReader(fragment))
	for {
		switch tokens.Next() {
		case markup.ErrorToken:
			return found, words.String()
		case markup.TextToken:
			if !hidden {
				words.Write(tokens.Text())
			}
		case markup.EndTagToken:
			hidden = false
		case markup.StartTagToken, markup.SelfClosingTagToken:
			name, more := tokens.TagName()
			hidden = string(name) == "script" || string(name) == "style"
			if string(name) == "img" {
				found = append(found, imageTag(tokens, more))
			}
		}
	}
}

func imageTag(tokens *markup.Tokenizer, more bool) picture {
	var image picture
	for more {
		var key, value []byte
		key, value, more = tokens.TagAttr()
		switch string(key) {
		case "src":
			image.address = string(value)
		case "alt":
			image.name = string(value)
		}
	}
	return image
}

func blank(words string) bool { return strings.TrimSpace(words) == "" }

// rentryMarks are rentry's centring arrows and colour spans, which carry no words
var rentryMarks = regexp.MustCompile(`->|<-|%[#\w]*%`)

func withoutRentryMarks(words string) string {
	return strings.Join(strings.Fields(rentryMarks.ReplaceAllString(words, " ")), " ")
}

func onlyLinks(paragraph ast.Node, source []byte) bool {
	links := 0
	for child := paragraph.FirstChild(); child != nil; child = child.NextSibling() {
		switch node := child.(type) {
		case *ast.Link:
			links++
		case *ast.Text:
			if !blank(string(node.Segment.Value(source))) {
				return false
			}
		default:
			return false
		}
	}
	return links > 0
}

func onlyAnchors(list ast.Node, source []byte) bool {
	anchors, others := 0, 0
	_ = ast.Walk(list, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering || node.Type() != ast.TypeInline {
			return ast.WalkContinue, nil
		}
		if link, ok := node.(*ast.Link); ok && strings.HasPrefix(string(link.Destination), "#") {
			anchors++
			return ast.WalkSkipChildren, nil
		}
		if words, ok := node.(*ast.Text); ok && strings.TrimSpace(string(words.Segment.Value(source))) == "" {
			return ast.WalkContinue, nil
		}
		others++
		return ast.WalkStop, nil
	})
	return anchors > 0 && others == 0
}

func plain(node ast.Node, source []byte) string {
	var written strings.Builder
	tagged := false
	_ = ast.Walk(node, func(child ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		switch inline := child.(type) {
		case *ast.Text:
			written.Write(util.UnescapePunctuations(inline.Segment.Value(source)))
			if inline.SoftLineBreak() {
				written.WriteByte(' ')
			}
		case *ast.String:
			written.Write(inline.Value)
		case *ast.RawHTML:
			tagged = true
			written.Write(inline.Segments.Value(source))
		}
		return ast.WalkContinue, nil
	})
	words := html.UnescapeString(written.String())
	if tagged {
		_, words = readHTML(written.String())
	}
	return strings.Join(strings.Fields(words), " ")
}

func firstOffset(node ast.Node, source []byte) (int, bool) {
	if code, ok := node.(*ast.FencedCodeBlock); ok {
		if code.Info != nil {
			return code.Info.Segment.Start, true
		}
		if code.Lines().Len() == 0 {
			return 0, false
		}
		return previousLine(source, code.Lines().At(0).Start), true
	}
	if node.Type() == ast.TypeBlock && node.Lines().Len() > 0 {
		return node.Lines().At(0).Start, true
	}
	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		if child.Type() != ast.TypeBlock {
			continue
		}
		if offset, ok := firstOffset(child, source); ok {
			return offset, true
		}
	}
	return 0, false
}

func lineStart(source []byte, offset int) int {
	for offset > 0 && source[offset-1] != '\n' {
		offset--
	}
	return offset
}

func previousLine(source []byte, offset int) int {
	start := lineStart(source, offset)
	if start == 0 {
		return 0
	}
	return lineStart(source, start-1)
}

var rule = regexp.MustCompile(`^ {0,3}(?:(?:-[ \t]*){3,}|(?:\*[ \t]*){3,}|(?:_[ \t]*){3,})$`)

func tidy(slice string) string {
	lines := strings.Split(strings.TrimRight(slice, " \t\n"), "\n")
	for len(lines) > 1 {
		last := lines[len(lines)-1]
		above := strings.TrimSpace(lines[len(lines)-2])
		if !rule.MatchString(last) || (above != "" && strings.Contains(last, "-")) {
			break
		}
		lines = strings.Split(strings.TrimRight(strings.Join(lines[:len(lines)-1], "\n"), " \t\n"), "\n")
	}
	return strings.Join(lines, "\n")
}

func containsEntry(images []readmeImage, entry string) bool {
	for _, image := range images {
		if image.Entry == entry {
			return true
		}
	}
	return false
}
