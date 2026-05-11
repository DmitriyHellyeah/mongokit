package mongokit

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/v2/bson"
)

func TestBaseField_DefaultID(t *testing.T) {
	t.Run("generates ID when zero", func(t *testing.T) {
		b := &BaseField{}
		assert.True(t, b.ID.IsZero(), "ID should be zero before DefaultID")

		b.DefaultID()

		assert.False(t, b.ID.IsZero(), "ID should be non-zero after DefaultID")
	})

	t.Run("does not overwrite existing ID", func(t *testing.T) {
		existing := bson.NewObjectID()
		b := &BaseField{ID: existing}

		b.DefaultID()

		assert.Equal(t, existing, b.ID, "DefaultID must not overwrite an existing ID")
	})
}

func TestBaseField_BeforeInsert(t *testing.T) {
	b := &BaseField{}

	before := time.Now()
	b.BeforeInsert()
	after := time.Now()

	assert.False(t, b.ID.IsZero(), "BeforeInsert must set a non-zero ID")
	assert.False(t, b.CreatedAt.IsZero(), "BeforeInsert must set CreatedAt")
	assert.False(t, b.UpdatedAt.IsZero(), "BeforeInsert must set UpdatedAt")

	assert.True(t, !b.CreatedAt.Before(before) && !b.CreatedAt.After(after),
		"CreatedAt should be within the test time window")
	assert.True(t, !b.UpdatedAt.Before(before) && !b.UpdatedAt.After(after),
		"UpdatedAt should be within the test time window")
}

func TestBaseField_BeforeUpdate(t *testing.T) {
	originalCreatedAt := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	b := &BaseField{
		CreatedAt: originalCreatedAt,
	}

	before := time.Now()
	b.BeforeUpdate()
	after := time.Now()

	assert.Equal(t, originalCreatedAt, b.CreatedAt,
		"BeforeUpdate must not modify CreatedAt")
	assert.False(t, b.UpdatedAt.IsZero(), "BeforeUpdate must set UpdatedAt")
	assert.True(t, !b.UpdatedAt.Before(before) && !b.UpdatedAt.After(after),
		"UpdatedAt should be within the test time window")
}

func TestBaseField_Validate(t *testing.T) {
	b := &BaseField{}
	assert.NoError(t, b.Validate(), "BaseField.Validate must return nil")
}
