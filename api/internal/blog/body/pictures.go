package body

import (
	"encoding/json"
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/google/uuid"
)

const (
	maxAlt          = 300
	maxCaption      = 300
	maxGalleryItems = 12
)

func (r *reader) image(path string, fields map[string]json.RawMessage) (Block, error) {
	if err := onlyKeys(path, fields, "type", "mediaId", "alt", "caption"); err != nil {
		return nil, err
	}
	return r.picture(path, fields)
}

func (r *reader) gallery(path string, fields map[string]json.RawMessage) (Block, error) {
	entries, err := entryList(path, fields, "A gallery holds a list of pictures.")
	if err != nil {
		return nil, err
	}
	if len(entries) == 0 {
		return nil, Problem{Path: path + ".content", Message: "A gallery needs at least one picture."}
	}
	if len(entries) > maxGalleryItems {
		return nil, Problem{
			Path:    path + ".content",
			Message: fmt.Sprintf("A gallery shows up to %d pictures.", maxGalleryItems),
		}
	}
	images := make([]Image, 0, len(entries))
	for index, entry := range entries {
		here := fmt.Sprintf("%s.content.%d", path, index)
		item, calloutType, err := r.node(here, entry)
		if err != nil {
			return nil, err
		}
		if calloutType != "galleryImage" {
			return nil, Problem{Path: here + ".type", Message: "A gallery holds gallery pictures."}
		}
		if err := onlyKeys(here, item, "type", "mediaId", "alt", "caption"); err != nil {
			return nil, err
		}
		picture, err := r.picture(here, item)
		if err != nil {
			return nil, err
		}
		images = append(images, picture)
	}
	return Gallery{Images: images}, nil
}

func (r *reader) picture(path string, fields map[string]json.RawMessage) (Image, error) {
	mediaID, err := readString(path+".mediaId", fields, "mediaId")
	if err != nil {
		return Image{}, err
	}
	named, err := uuid.Parse(mediaID)
	if err != nil {
		return Image{}, Problem{
			Path: path + ".mediaId",
			Message: "A picture names an image uploaded to this post. " +
				"Upload the file first and place the id it answers with.",
		}
	}
	alt, err := readString(path+".alt", fields, "alt")
	if err != nil {
		return Image{}, err
	}
	alt = strings.TrimSpace(alt)
	if alt == "" {
		return Image{}, Problem{
			Path:    path + ".alt",
			Message: "Describe the picture for a reader who cannot see it.",
		}
	}
	if err := plainText(path+".alt", alt, maxAlt); err != nil {
		return Image{}, err
	}
	caption := ""
	if _, carried := fields["caption"]; carried {
		if caption, err = readString(path+".caption", fields, "caption"); err != nil {
			return Image{}, err
		}
		caption = strings.TrimSpace(caption)
		if err := plainText(path+".caption", caption, maxCaption); err != nil {
			return Image{}, err
		}
	}
	r.text += len(alt) + len(caption)
	if r.text > maxDocumentText {
		return Image{}, Problem{Path: path, Message: "This post is too long."}
	}
	return Image{MediaID: named.String(), Alt: alt, Caption: caption}, nil
}

func plainText(path, value string, limit int) error {
	if !utf8.ValidString(value) {
		return Problem{Path: path, Message: "This text is not valid UTF-8."}
	}
	if len([]rune(value)) > limit {
		return Problem{
			Path:    path,
			Message: fmt.Sprintf("Keep this to %d characters or fewer.", limit),
		}
	}
	if strings.ContainsFunc(value, unicode.IsControl) {
		return Problem{Path: path, Message: "Use plain text, on one line."}
	}
	return nil
}

func (d Document) MediaIDs() []string {
	found := make([]string, 0, 4)
	for _, block := range d.Blocks {
		switch shape := block.(type) {
		case Image:
			found = append(found, shape.MediaID)
		case Gallery:
			for _, picture := range shape.Images {
				found = append(found, picture.MediaID)
			}
		}
	}
	return found
}
