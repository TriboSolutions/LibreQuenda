package platform

import (
	"context"
	"database/sql"
	"errors"
	"librequenda/model"
	"librequenda/shared"
	"net/http"

	"github.com/google/uuid"
)

// TagCatalogue is every category with its tags: public (the filters use it) and
// what the superadministrator's editor shows after each change.
func (s *platformService) TagCatalogue(ctx context.Context) (*TagCatalogueResponse, *shared.ErrorResponse) {
	var queries = model.New(s.db)
	var categories, err = queries.ListTagCategories(ctx)
	if err != nil {
		return nil, failed("errors.tags_load_failed")
	}
	var tags []model.ListTagsRow
	tags, err = queries.ListTags(ctx)
	if err != nil {
		return nil, failed("errors.tags_load_failed")
	}
	var byCategory = map[uuid.UUID][]TagDto{}
	for _, t := range tags {
		byCategory[t.CategoryID] = append(byCategory[t.CategoryID], TagDto{Id: t.ID.String(), Slug: t.Slug, Name: t.Name, FacilityCount: t.FacilityCount})
	}
	var response = &TagCatalogueResponse{Categories: make([]TagCategoryDto, 0, len(categories))}
	for _, c := range categories {
		response.Categories = append(response.Categories, TagCategoryDto{
			Id:       c.ID.String(),
			Name:     c.Name,
			Position: c.Position,
			Tags:     append([]TagDto{}, byCategory[c.ID]...),
		})
	}
	return response, nil
}

func (s *platformService) CreateTagCategory(ctx context.Context, input TagCategoryInput) (*TagCatalogueResponse, *shared.ErrorResponse) {
	if key := input.validate(); key != "" {
		return nil, &shared.ErrorResponse{Code: http.StatusUnprocessableEntity, Message: key}
	}
	var queries = model.New(s.db)
	var position, err = queries.NextTagCategoryPosition(ctx)
	if err == nil {
		_, err = queries.CreateTagCategory(ctx, model.CreateTagCategoryParams{Name: input.Name, Position: position})
	}
	if isUniqueViolation(err) {
		return nil, &shared.ErrorResponse{Code: http.StatusConflict, Message: "errors.tag_category_taken"}
	}
	if err != nil {
		return nil, failed("errors.tags_save_failed")
	}
	return s.TagCatalogue(ctx)
}

func (s *platformService) UpdateTagCategory(ctx context.Context, id uuid.UUID, input TagCategoryInput) (*TagCatalogueResponse, *shared.ErrorResponse) {
	if key := input.validate(); key != "" {
		return nil, &shared.ErrorResponse{Code: http.StatusUnprocessableEntity, Message: key}
	}
	var queries = model.New(s.db)
	var categories, err = queries.ListTagCategories(ctx)
	if err != nil {
		return nil, failed("errors.tags_save_failed")
	}
	var position = int32(-1)
	for _, c := range categories {
		if c.ID == id {
			position = c.Position
		}
	}
	if position < 0 {
		return nil, &shared.ErrorResponse{Code: http.StatusNotFound, Message: "errors.tag_category_not_found"}
	}
	if input.Position != nil {
		position = *input.Position
	}
	_, err = queries.UpdateTagCategory(ctx, model.UpdateTagCategoryParams{ID: id, Name: input.Name, Position: position})
	if isUniqueViolation(err) {
		return nil, &shared.ErrorResponse{Code: http.StatusConflict, Message: "errors.tag_category_taken"}
	}
	if err != nil {
		return nil, failed("errors.tags_save_failed")
	}
	return s.TagCatalogue(ctx)
}

func (s *platformService) DeleteTagCategory(ctx context.Context, id uuid.UUID) (*TagCatalogueResponse, *shared.ErrorResponse) {
	var deleted, err = model.New(s.db).DeleteTagCategory(ctx, id)
	if err != nil {
		return nil, failed("errors.tags_save_failed")
	}
	if deleted == 0 {
		return nil, &shared.ErrorResponse{Code: http.StatusNotFound, Message: "errors.tag_category_not_found"}
	}
	return s.TagCatalogue(ctx)
}

func (s *platformService) CreateTag(ctx context.Context, input TagInput) (*TagCatalogueResponse, *shared.ErrorResponse) {
	if key := input.validate(true); key != "" {
		return nil, &shared.ErrorResponse{Code: http.StatusUnprocessableEntity, Message: key}
	}
	var categoryId, parseErr = uuid.Parse(input.CategoryId)
	if parseErr != nil {
		return nil, &shared.ErrorResponse{Code: http.StatusNotFound, Message: "errors.tag_category_not_found"}
	}
	var _, err = model.New(s.db).CreateTag(ctx, model.CreateTagParams{CategoryID: categoryId, Slug: input.Slug, Name: input.Name})
	return s.afterTagSave(ctx, err)
}

func (s *platformService) UpdateTag(ctx context.Context, id uuid.UUID, input TagInput) (*TagCatalogueResponse, *shared.ErrorResponse) {
	if key := input.validate(false); key != "" {
		return nil, &shared.ErrorResponse{Code: http.StatusUnprocessableEntity, Message: key}
	}
	var categoryId, parseErr = uuid.Parse(input.CategoryId)
	if parseErr != nil {
		return nil, &shared.ErrorResponse{Code: http.StatusNotFound, Message: "errors.tag_category_not_found"}
	}
	var updated, err = model.New(s.db).UpdateTag(ctx, model.UpdateTagParams{ID: id, CategoryID: categoryId, Name: input.Name})
	if err == nil && updated == 0 {
		return nil, &shared.ErrorResponse{Code: http.StatusNotFound, Message: "errors.tag_not_found"}
	}
	return s.afterTagSave(ctx, err)
}

func (s *platformService) DeleteTag(ctx context.Context, id uuid.UUID) (*TagCatalogueResponse, *shared.ErrorResponse) {
	var deleted, err = model.New(s.db).DeleteTag(ctx, id)
	if err != nil {
		return nil, failed("errors.tags_save_failed")
	}
	if deleted == 0 {
		return nil, &shared.ErrorResponse{Code: http.StatusNotFound, Message: "errors.tag_not_found"}
	}
	return s.TagCatalogue(ctx)
}

// afterTagSave answers a tag insert or update: a taken name or slug, a category
// that does not exist, or the catalogue as it is now.
func (s *platformService) afterTagSave(ctx context.Context, err error) (*TagCatalogueResponse, *shared.ErrorResponse) {
	if isUniqueViolation(err) {
		return nil, &shared.ErrorResponse{Code: http.StatusConflict, Message: "errors.tag_taken"}
	}
	if isForeignKeyViolation(err) || errors.Is(err, sql.ErrNoRows) {
		return nil, &shared.ErrorResponse{Code: http.StatusNotFound, Message: "errors.tag_category_not_found"}
	}
	if err != nil {
		return nil, failed("errors.tags_save_failed")
	}
	return s.TagCatalogue(ctx)
}
