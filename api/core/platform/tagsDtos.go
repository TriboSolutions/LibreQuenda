package platform

import (
	"librequenda/validate"
	"strings"
)

// TagDto is one tag of the catalogue; Slug is what facilities and addresses use.
type TagDto struct {
	Id            string `json:"id"`
	Slug          string `json:"slug"`
	Name          string `json:"name"`
	FacilityCount int32  `json:"facility_count"` // facilities that carry it, in every council
}

type TagCategoryDto struct {
	Id       string   `json:"id"`
	Name     string   `json:"name"`
	Position int32    `json:"position"`
	Tags     []TagDto `json:"tags"` // by name
}

type TagCatalogueResponse struct {
	Categories []TagCategoryDto `json:"categories"` // in their order
}

// maxTagCategoryPosition: far more categories than a catalogue has, so the order stays small numbers.
const maxTagCategoryPosition = 1000

type TagCategoryInput struct {
	Name     string `json:"name"`
	Position *int32 `json:"position"` // update only: nil keeps it
}

func (in *TagCategoryInput) validate() string {
	in.Name = strings.Join(strings.Fields(in.Name), " ")
	if in.Name == "" || len([]rune(in.Name)) > 60 {
		return "errors.invalid_tag_category_name"
	}
	if in.Position != nil && (*in.Position < 0 || *in.Position > maxTagCategoryPosition) {
		return "errors.invalid_tag_category_position"
	}
	return ""
}

type TagInput struct {
	CategoryId string `json:"category_id"`
	Name       string `json:"name"`
	Slug       string `json:"slug"` // create only; it never changes after
}

func (in *TagInput) validate(creating bool) string {
	in.Name = strings.Join(strings.Fields(in.Name), " ")
	in.Slug = strings.ToLower(strings.TrimSpace(in.Slug))
	switch {
	case in.Name == "" || len([]rune(in.Name)) > 40:
		return "errors.invalid_tag_name"
	case creating && !validate.Slug(in.Slug):
		return "errors.invalid_slug"
	}
	return ""
}
