package mongokit

import (
	"context"
	"slices"

	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

const (
	MinPaginationLimit     = 1
	MaxPaginationLimit     = 100
	DefaultPaginationLimit = 20
)

// CursorDirection defines the pagination direction.
type CursorDirection int

const (
	CursorNext CursorDirection = iota
	CursorPrev
)

// CursorPagination holds parameters for cursor-based pagination queries.
type CursorPagination struct {
	Cursor    string          `json:"cursor" bson:"cursor"`
	Direction CursorDirection `json:"direction" bson:"direction"`
	Limit     int64           `json:"limit" bson:"limit"`
}

// DefaultCursorPagination returns a CursorPagination with default values.
func DefaultCursorPagination() *CursorPagination {
	return &CursorPagination{
		Direction: CursorNext,
		Limit:     DefaultPaginationLimit,
	}
}

// CursorPaginationResult holds the result of a cursor-based paginated query.
type CursorPaginationResult[T Document] struct {
	Items      []T    `json:"items"`
	HasNext    bool   `json:"hasNext"`
	HasPrev    bool   `json:"hasPrev"`
	NextCursor string `json:"nextCursor,omitempty"`
	PrevCursor string `json:"prevCursor,omitempty"`
}

func clampLimit(n int64) int64 {
	if n < MinPaginationLimit {
		return MinPaginationLimit
	}
	if n > MaxPaginationLimit {
		return MaxPaginationLimit
	}
	return n
}

// FindPaginated returns a page of documents using offset-based pagination.
// Clamps page and pageSize to valid range (min: 1, max: MaxPaginationLimit).
func (r *Repository[T]) FindPaginated(ctx context.Context, filter any, page, pageSize int64) ([]T, error) {
	if page < 1 {
		page = 1
	}
	pageSize = clampLimit(pageSize)

	opts := options.Find().
		SetSkip((page - 1) * pageSize).
		SetLimit(pageSize)
	return r.FindDecoded(ctx, filter, opts)
}

// FindPaginatedWithTotal returns a page of documents and the total count.
func (r *Repository[T]) FindPaginatedWithTotal(ctx context.Context, filter any, page, pageSize int64) ([]T, int64, error) {
	total, err := r.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	results, err := r.FindPaginated(ctx, filter, page, pageSize)
	if err != nil {
		return nil, 0, err
	}

	return results, total, nil
}

// FindCursorPaginated performs cursor-based pagination.
// Filter, sort and cursor condition ($gt/$lt) are the caller's responsibility.
// Fetches limit+1 to detect whether more pages exist.
// For CursorPrev direction, results are reversed to maintain natural order.
// cursorEncoder converts a document into a cursor string for the response.
func (r *Repository[T]) FindCursorPaginated(
	ctx context.Context,
	filter any,
	pagination *CursorPagination,
	sort any,
	cursorEncoder func(T) (string, error),
) (*CursorPaginationResult[T], error) {
	if pagination == nil {
		pagination = DefaultCursorPagination()
	}
	limit := clampLimit(pagination.Limit)

	opts := options.Find().SetLimit(limit + 1)
	if sort != nil {
		opts.SetSort(sort)
	}

	items, err := r.FindDecoded(ctx, filter, opts)
	if err != nil {
		return nil, err
	}

	hasMore := int64(len(items)) > limit
	if hasMore {
		items = items[:limit]
	}

	if pagination.Direction == CursorPrev {
		slices.Reverse(items)
	}

	result := &CursorPaginationResult[T]{
		Items: items,
	}

	if len(items) == 0 {
		return result, nil
	}

	firstItem := items[0]
	lastItem := items[len(items)-1]

	switch pagination.Direction {
	case CursorNext:
		result.HasNext = hasMore
		result.HasPrev = pagination.Cursor != ""
		if result.HasNext {
			if result.NextCursor, err = cursorEncoder(lastItem); err != nil {
				return nil, err
			}
		}
		if result.HasPrev {
			if result.PrevCursor, err = cursorEncoder(firstItem); err != nil {
				return nil, err
			}
		}
	case CursorPrev:
		result.HasPrev = hasMore
		result.HasNext = pagination.Cursor != ""
		if result.HasPrev {
			if result.PrevCursor, err = cursorEncoder(firstItem); err != nil {
				return nil, err
			}
		}
		if result.HasNext {
			if result.NextCursor, err = cursorEncoder(lastItem); err != nil {
				return nil, err
			}
		}
	}

	return result, nil
}
