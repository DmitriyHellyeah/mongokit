package mongokit

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

// ---------------------------------------------------------------------------
// SplitSortField
// ---------------------------------------------------------------------------

func TestSplitSortField(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantKey   string
		wantSort  int32
	}{
		{name: "plain field", input: "email", wantKey: "email", wantSort: 1},
		{name: "explicit ascending prefix", input: "+email", wantKey: "email", wantSort: 1},
		{name: "descending prefix", input: "-createdAt", wantKey: "createdAt", wantSort: -1},
		{name: "empty string", input: "", wantKey: "", wantSort: 1},
		{name: "dash only", input: "-", wantKey: "", wantSort: -1},
		{name: "whitespace trimmed", input: "  email  ", wantKey: "email", wantSort: 1},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			key, sort := SplitSortField(tc.input)
			assert.Equal(t, tc.wantKey, key, "key mismatch for input %q", tc.input)
			assert.Equal(t, tc.wantSort, sort, "sort mismatch for input %q", tc.input)
		})
	}
}

// ---------------------------------------------------------------------------
// UniqueIndexes
// ---------------------------------------------------------------------------

func TestUniqueIndexes(t *testing.T) {
	t.Run("single field", func(t *testing.T) {
		models := UniqueIndexes("email")
		require.Len(t, models, 1)
		require.NotNil(t, models[0].Options)

		keys, ok := models[0].Keys.(bson.D)
		require.True(t, ok, "keys should be bson.D")
		require.Len(t, keys, 1)
		assert.Equal(t, "email", keys[0].Key)
		assert.Equal(t, int32(1), keys[0].Value)
	})

	t.Run("composite index with descending field", func(t *testing.T) {
		models := UniqueIndexes("userId,-createdAt")
		require.Len(t, models, 1)

		keys, ok := models[0].Keys.(bson.D)
		require.True(t, ok, "keys should be bson.D")
		require.Len(t, keys, 2)
		assert.Equal(t, "userId", keys[0].Key)
		assert.Equal(t, int32(1), keys[0].Value)
		assert.Equal(t, "createdAt", keys[1].Key)
		assert.Equal(t, int32(-1), keys[1].Value)
	})

	t.Run("empty args", func(t *testing.T) {
		models := UniqueIndexes()
		assert.Empty(t, models)
	})
}

// ---------------------------------------------------------------------------
// NonUniqueIndexes
// ---------------------------------------------------------------------------

func TestNonUniqueIndexes(t *testing.T) {
	t.Run("single field", func(t *testing.T) {
		models := NonUniqueIndexes("status")
		require.Len(t, models, 1)
		assert.Nil(t, models[0].Options, "non-unique index should have nil options")

		keys, ok := models[0].Keys.(bson.D)
		require.True(t, ok)
		require.Len(t, keys, 1)
		assert.Equal(t, "status", keys[0].Key)
		assert.Equal(t, int32(1), keys[0].Value)
	})

	t.Run("composite index", func(t *testing.T) {
		models := NonUniqueIndexes("-score,userId")
		require.Len(t, models, 1)

		keys, ok := models[0].Keys.(bson.D)
		require.True(t, ok)
		require.Len(t, keys, 2)
		assert.Equal(t, "score", keys[0].Key)
		assert.Equal(t, int32(-1), keys[0].Value)
		assert.Equal(t, "userId", keys[1].Key)
		assert.Equal(t, int32(1), keys[1].Value)
	})

	t.Run("empty args", func(t *testing.T) {
		models := NonUniqueIndexes()
		assert.Empty(t, models)
	})
}

// ---------------------------------------------------------------------------
// BuildIndexes
// ---------------------------------------------------------------------------

func TestBuildIndexes(t *testing.T) {
	t.Run("merges two groups", func(t *testing.T) {
		unique := UniqueIndexes("email")
		nonUnique := NonUniqueIndexes("status", "createdAt")

		result := BuildIndexes(unique, nonUnique)
		assert.Len(t, result, 3)
	})

	t.Run("empty groups return empty slice", func(t *testing.T) {
		result := BuildIndexes([]mongo.IndexModel{}, []mongo.IndexModel{})
		assert.Empty(t, result)
	})

	t.Run("nil groups are ignored", func(t *testing.T) {
		result := BuildIndexes(nil, nil)
		assert.Empty(t, result)
	})
}

// ---------------------------------------------------------------------------
// MapNotFoundErr
// ---------------------------------------------------------------------------

func TestMapNotFoundErr(t *testing.T) {
	customErr := errors.New("document not found")

	tests := []struct {
		name    string
		err     error
		wantErr error
	}{
		{
			name:    "mongo.ErrNoDocuments is replaced with custom error",
			err:     mongo.ErrNoDocuments,
			wantErr: customErr,
		},
		{
			name:    "nil returns nil",
			err:     nil,
			wantErr: nil,
		},
		{
			name:    "other error is returned unchanged",
			err:     errors.New("some other error"),
			wantErr: errors.New("some other error"),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := MapNotFoundErr(tc.err, customErr)
			if tc.wantErr == nil {
				assert.NoError(t, got)
			} else {
				require.Error(t, got)
				assert.Equal(t, tc.wantErr.Error(), got.Error())
			}
		})
	}
}

// ---------------------------------------------------------------------------
// prepareUpdate
// ---------------------------------------------------------------------------

type updateTestDoc struct {
	BaseField `bson:",inline"`
	Name      string `bson:"name"`
}

func (*updateTestDoc) CollectionName() string { return "update_test" }

func TestPrepareUpdate(t *testing.T) {
	t.Run("nil input returns ErrNilUpdate", func(t *testing.T) {
		result, err := prepareUpdate(nil)
		assert.Nil(t, result)
		assert.ErrorIs(t, err, ErrNilUpdate)
	})

	t.Run("bson.M with dollar key passes through unchanged", func(t *testing.T) {
		input := bson.M{"$set": bson.M{"name": "x"}}
		result, err := prepareUpdate(input)
		require.NoError(t, err)
		assert.Equal(t, input, result)
	})

	t.Run("struct is wrapped in $set", func(t *testing.T) {
		doc := updateTestDoc{Name: "Alice"}
		result, err := prepareUpdate(doc)
		require.NoError(t, err)

		m, ok := result.(bson.M)
		require.True(t, ok, "result should be bson.M")
		_, hasSet := m["$set"]
		assert.True(t, hasSet, "struct payload must be wrapped in $set")
	})

	t.Run("unsupported type returns ErrUnsupportedUpdateType", func(t *testing.T) {
		result, err := prepareUpdate("invalid")
		assert.Nil(t, result)
		assert.ErrorIs(t, err, ErrUnsupportedUpdateType)
	})
}
