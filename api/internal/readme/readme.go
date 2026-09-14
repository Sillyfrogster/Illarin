// Package readme divides a README into the titled sections that seed an asset page.
package readme

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

// Image is a picture held in the archive, named by the words written for it.
type Image struct {
	Entry string
	Name  string
}

// Remote is a picture the README shows from elsewhere, which Illarin lists by its address and never fetches.
type Remote struct {
	Address string
	Name    string
}

// Section is one titled part of a README, its writing kept as the author wrote it.
type Section struct {
	Title  string
	Text   string
	Images []Image
	Remote []Remote
}

// Page is what a README seeds, a cover, the untitled opening and the sections after it.
type Page struct {
	Cover    *Image
	Opening  Section
	Sections []Section
}

// Find names the archive entry an image address refers to, and says whether the archive holds it.
type Find func(address string) (string, bool)

var parser = goldmark.New(goldmark.WithExtensions(extension.Table)).Parser()

// Read divides a README at its shallowest heading below the title and moves each picture the archive holds beside its section.
func Read(source string, find Find) Page {
	source = strings.TrimPrefix(strings.ReplaceAll(source, "\r\n", "\n"), "\uFEFF")
	raw := []byte(source)
	parts := divide(parser.Parse(text.NewReader(raw)), raw, find)
	title := titlePart(parts)
	depth := sectionDepth(parts, title)

	page := Page{}
	current := &page.Opening
	for index, part := range parts {
		switch {
		case index == title:
		case part.depth > 0 && part.depth == depth:
			page.Sections = append(page.Sections, Section{Title: part.words})
			current = &page.Sections[len(page.Sections)-1]
		default:
			current.add(part)
		}
	}
	page.Sections = kept(page.Sections, parts, title, depth)
	if len(page.Opening.Images) > 0 {
		cover := page.Opening.Images[0]
		page.Cover = &cover
		page.Opening.Images = page.Opening.Images[1:]
	}
	return page
}

// InArchive finds an address beside the README, or from the repository root when it starts with a slash, and never outside the root.
func InArchive(root, folder string, has func(entry string) bool) Find {
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
	images   []Image
	remote   []Remote
	pictures bool
	anchors  bool
}

func (s *Section) add(p part) {
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

// divide slices the source at the first line of every top-level node, so each part keeps the author's own writing.
func divide(document ast.Node, source []byte, find Find) []part {
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

func (p *part) read(source []byte, find Find) {
	switch node := p.node.(type) {
	case *ast.Heading:
		p.depth = node.Level
		p.words = plain(node, source)
	case *ast.Paragraph:
		found, only := paragraphPictures(node, source)
		p.images, p.remote = located(found, find)
		p.pictures = only
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

// titlePart finds the heading that names the whole README, the first words when no other heading sits as high.
func titlePart(parts []part) int {
	for index, candidate := range parts {
		if candidate.pictures {
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

// kept leaves out sections with nothing in them and a contents list whose links only point within the README.
func kept(sections []Section, parts []part, title, depth int) []Section {
	anchorsOnly := make([]bool, 0, len(sections))
	for index, candidate := range parts {
		switch {
		case index == title:
		case candidate.depth > 0 && candidate.depth == depth:
			anchorsOnly = append(anchorsOnly, true)
		case len(anchorsOnly) > 0 && !candidate.pictures:
			last := len(anchorsOnly) - 1
			anchorsOnly[last] = anchorsOnly[last] && candidate.anchors
		}
	}
	result := make([]Section, 0, len(sections))
	for index, section := range sections {
		pictured := len(section.Images) > 0 || len(section.Remote) > 0
		if (section.Text != "" || pictured) && !(anchorsOnly[index] && !pictured) {
			result = append(result, section)
		}
	}
	return result
}

type picture struct {
	address string
	name    string
}

// located sorts pictures into those the archive holds and those shown from another site.
func located(found []picture, find Find) ([]Image, []Remote) {
	var images []Image
	var remote []Remote
	for _, candidate := range found {
		if entry, ok := find(candidate.address); ok {
			if !containsEntry(images, entry) {
				images = append(images, Image{Entry: entry, Name: candidate.name})
			}
			continue
		}
		if shown, ok := remotePicture(candidate); ok && !slices.Contains(remote, shown) {
			remote = append(remote, shown)
		}
	}
	return images, remote
}

// badgeHosts serve the little status badges a README wears, which are not pictures of the extension.
var badgeHosts = map[string]bool{"img.shields.io": true, "badgen.net": true, "badge.fury.io": true}

// remotePicture keeps a picture served from another site when it is one Illarin could hold a copy of.
func remotePicture(candidate picture) (Remote, bool) {
	address := strings.TrimSpace(candidate.address)
	parsed, err := url.Parse(address)
	if err != nil || (parsed.Scheme != "https" && parsed.Scheme != "http") || parsed.Host == "" {
		return Remote{}, false
	}
	if badgeHosts[parsed.Host] || strings.EqualFold(path.Ext(parsed.Path), ".svg") {
		return Remote{}, false
	}
	return Remote{Address: address, Name: candidate.name}, true
}

// paragraphPictures reads the pictures in a paragraph and says whether pictures are all it holds.
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

// tablePictures reads the pictures in a table and says whether its rows hold nothing else, the header naming any picture without words of its own.
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

// readHTML reads the pictures in a piece of HTML and the words a reader would see in it.
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

// plain reads a heading or picture's words, leaving out markup and what a script or style holds.
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

// firstOffset finds where a node's writing starts, taking a fenced code block from its opening fence.
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

// tidy drops the blank lines and dividing rules a part trails, keeping a heading's underline.
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

func containsEntry(images []Image, entry string) bool {
	for _, image := range images {
		if image.Entry == entry {
			return true
		}
	}
	return false
}
