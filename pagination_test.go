package mongokit

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/v2/bson"
)

func seedManyUsers(t *testing.T, repo *Repository[*TestUser], count int) []*TestUser {
	t.Helper()
	ctx := context.Background()
	var users []*TestUser
	for i := 0; i < count; i++ {
		u, err := repo.InsertOne(ctx, &TestUser{
			Name:   fmt.Sprintf("User %03d", i),
			Email:  fmt.Sprintf("user%03d@example.com", i),
			Status: "active",
		})
		require.NoError(t, err)
		users = append(users, u)
	}
	return users
}

func testCursorEncoder(u *TestUser) (string, error) {
	return u.ID.Hex(), nil
}

func TestFindPaginated(t *testing.T) {
	repo, _, cleanup := setupTestRepo(t)
	defer cleanup()

	seedManyUsers(t, repo, 25)
	ctx := context.Background()

	t.Run("page 1 size 10 returns 10 items", func(t *testing.T) {
		items, err := repo.FindPaginated(ctx, bson.M{}, 1, 10)
		require.NoError(t, err)
		assert.Len(t, items, 10)
	})

	t.Run("page 3 size 10 returns 5 items", func(t *testing.T) {
		items, err := repo.FindPaginated(ctx, bson.M{}, 3, 10)
		require.NoError(t, err)
		assert.Len(t, items, 5)
	})
}

func TestFindPaginated_Clamping(t *testing.T) {
	repo, _, cleanup := setupTestRepo(t)
	defer cleanup()

	seedManyUsers(t, repo, 5)
	ctx := context.Background()

	t.Run("page less than 1 clamped to 1", func(t *testing.T) {
		items, err := repo.FindPaginated(ctx, bson.M{}, 0, 10)
		require.NoError(t, err)
		assert.Len(t, items, 5)
	})

	t.Run("pageSize less than 1 clamped to 1", func(t *testing.T) {
		items, err := repo.FindPaginated(ctx, bson.M{}, 1, 0)
		require.NoError(t, err)
		assert.Len(t, items, 1)
	})

	t.Run("pageSize greater than 100 clamped to 100", func(t *testing.T) {
		items, err := repo.FindPaginated(ctx, bson.M{}, 1, 200)
		require.NoError(t, err)
		assert.Len(t, items, 5)
	})
}

func TestFindPaginatedWithTotal(t *testing.T) {
	repo, _, cleanup := setupTestRepo(t)
	defer cleanup()

	seedManyUsers(t, repo, 25)
	ctx := context.Background()

	t.Run("page 1 size 10 returns 10 items and total 25", func(t *testing.T) {
		items, total, err := repo.FindPaginatedWithTotal(ctx, bson.M{}, 1, 10)
		require.NoError(t, err)
		assert.Len(t, items, 10)
		assert.Equal(t, int64(25), total)
	})
}

func TestFindCursorPaginated_Next(t *testing.T) {
	repo, _, cleanup := setupTestRepo(t)
	defer cleanup()

	users := seedManyUsers(t, repo, 25)
	ctx := context.Background()

	t.Run("first page has 10 items and HasNext", func(t *testing.T) {
		page, err := repo.FindCursorPaginated(ctx,
			bson.M{},
			&CursorPagination{Limit: 10, Direction: CursorNext},
			bson.D{{Key: "_id", Value: 1}},
			testCursorEncoder,
		)
		require.NoError(t, err)
		assert.Len(t, page.Items, 10)
		assert.True(t, page.HasNext)
		assert.NotEmpty(t, page.NextCursor)
	})

	t.Run("second page from 10th user", func(t *testing.T) {
		page, err := repo.FindCursorPaginated(ctx,
			bson.M{"_id": bson.M{"$gt": users[9].ID}},
			&CursorPagination{Cursor: users[9].ID.Hex(), Limit: 10, Direction: CursorNext},
			bson.D{{Key: "_id", Value: 1}},
			testCursorEncoder,
		)
		require.NoError(t, err)
		assert.Len(t, page.Items, 10)
		assert.True(t, page.HasNext)
		assert.True(t, page.HasPrev)
	})

	t.Run("third page has 5 items and HasNext false", func(t *testing.T) {
		page, err := repo.FindCursorPaginated(ctx,
			bson.M{"_id": bson.M{"$gt": users[19].ID}},
			&CursorPagination{Cursor: users[19].ID.Hex(), Limit: 10, Direction: CursorNext},
			bson.D{{Key: "_id", Value: 1}},
			testCursorEncoder,
		)
		require.NoError(t, err)
		assert.Len(t, page.Items, 5)
		assert.False(t, page.HasNext)
	})
}

func TestFindCursorPaginated_Prev(t *testing.T) {
	repo, _, cleanup := setupTestRepo(t)
	defer cleanup()

	users := seedManyUsers(t, repo, 25)
	ctx := context.Background()

	t.Run("navigate backward from user 10", func(t *testing.T) {
		page, err := repo.FindCursorPaginated(ctx,
			bson.M{"_id": bson.M{"$lt": users[10].ID}},
			&CursorPagination{Cursor: users[10].ID.Hex(), Limit: 10, Direction: CursorPrev},
			bson.D{{Key: "_id", Value: -1}},
			testCursorEncoder,
		)
		require.NoError(t, err)
		assert.Len(t, page.Items, 10)
		assert.True(t, page.HasNext)
	})
}

func TestFindCursorPaginated_Clamping(t *testing.T) {
	repo, _, cleanup := setupTestRepo(t)
	defer cleanup()

	seedManyUsers(t, repo, 5)
	ctx := context.Background()

	t.Run("limit less than 1 clamped", func(t *testing.T) {
		page, err := repo.FindCursorPaginated(ctx,
			bson.M{},
			&CursorPagination{Limit: 0, Direction: CursorNext},
			bson.D{{Key: "_id", Value: 1}},
			testCursorEncoder,
		)
		require.NoError(t, err)
		assert.Len(t, page.Items, MinPaginationLimit)
	})

	t.Run("limit greater than 100 clamped", func(t *testing.T) {
		page, err := repo.FindCursorPaginated(ctx,
			bson.M{},
			&CursorPagination{Limit: 200, Direction: CursorNext},
			bson.D{{Key: "_id", Value: 1}},
			testCursorEncoder,
		)
		require.NoError(t, err)
		assert.Len(t, page.Items, 5)
		assert.False(t, page.HasNext)
	})

	t.Run("nil pagination uses defaults", func(t *testing.T) {
		page, err := repo.FindCursorPaginated(ctx,
			bson.M{},
			nil,
			bson.D{{Key: "_id", Value: 1}},
			testCursorEncoder,
		)
		require.NoError(t, err)
		assert.Len(t, page.Items, 5)
	})
}
