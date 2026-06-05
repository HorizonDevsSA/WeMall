package service

import (
	"context"
	"fmt"
	"sort"

	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/structpb"

	productv1 "github.com/wemall/gen/product/v1"
	"github.com/wemall/product-service/internal/db"
)

// CategoryService handles category tree operations.
type CategoryService struct {
	q *db.Queries
}

func NewCategoryService(q *db.Queries) *CategoryService {
	return &CategoryService{q: q}
}

// ListCategories returns the full active category tree in the requested language.
// The tree is built in-memory from a flat SQL result ordered by level then sort_order.
func (s *CategoryService) ListCategories(ctx context.Context, lang string) ([]*productv1.Category, error) {
	rows, err := s.q.ListCategories(ctx, lang)
	if err != nil {
		return nil, fmt.Errorf("list categories: %w", err)
	}
	flat := make([]db.CategoryWithTranslation, len(rows))
	for i, r := range rows {
		flat[i] = db.CategoryWithTranslation(r)
	}
	return buildCategoryTree(flat), nil
}

// GetCategory returns a single category by slug with its translated name.
func (s *CategoryService) GetCategory(ctx context.Context, slugStr, lang string) (*productv1.Category, error) {
	row, err := s.q.GetCategoryBySlug(ctx, db.GetCategoryBySlugParams{
		Language: lang,
		Slug:     slugStr,
	})
	if err != nil {
		return nil, fmt.Errorf("category not found: %w", err)
	}
	c := db.CategoryWithTranslation(row)
	return mapCategory(&c), nil
}

// GetCategoryByID returns a single category by UUID (used internally for validation).
func (s *CategoryService) GetCategoryByID(ctx context.Context, id uuid.UUID, lang string) (*productv1.Category, error) {
	row, err := s.q.GetCategoryByID(ctx, db.GetCategoryByIDParams{
		Language: lang,
		ID:       id,
	})
	if err != nil {
		return nil, fmt.Errorf("category not found: %w", err)
	}
	c := db.CategoryWithTranslation(row)
	return mapCategory(&c), nil
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func buildCategoryTree(flat []db.CategoryWithTranslation) []*productv1.Category {
	catMap := make(map[string]*productv1.Category, len(flat))
	var roots []*productv1.Category

	for i := range flat {
		node := mapCategory(&flat[i])
		catMap[node.Id] = node
	}

	for i := range flat {
		idStr := flat[i].ID.String()
		node := catMap[idStr]
		if flat[i].ParentID == nil {
			roots = append(roots, node)
		} else {
			pIDStr := flat[i].ParentID.String()
			if parent, ok := catMap[pIDStr]; ok {
				parent.Children = append(parent.Children, node)
			} else {
				roots = append(roots, node)
			}
		}
	}

	sortTree(roots)
	return roots
}

func sortTree(nodes []*productv1.Category) {
	sort.Slice(nodes, func(i, j int) bool {
		return nodes[i].SortOrder < nodes[j].SortOrder
	})
	for _, n := range nodes {
		if len(n.Children) > 0 {
			sortTree(n.Children)
		}
	}
}

func mapCategory(c *db.CategoryWithTranslation) *productv1.Category {
	return &productv1.Category{
		Id:              c.ID.String(),
		ParentId:        getParentIDString(c.ParentID),
		Name:            c.Name,
		Slug:            c.Slug,
		IconUrl:         getVal(c.IconUrl),
		BannerUrl:       getVal(c.BannerUrl),
		Level:           c.Level,
		AttributeSchema: bytesToStruct(c.AttributeSchema),
		SortOrder:       c.SortOrder,
		Children:        []*productv1.Category{},
	}
}

func bytesToStruct(b []byte) *structpb.Struct {
	if len(b) == 0 {
		s, _ := structpb.NewStruct(nil)
		return s
	}
	return jsonToStruct(b)
}
