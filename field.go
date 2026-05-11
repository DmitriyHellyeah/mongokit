package mongokit

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// BaseField represents a structure with default fields for MongoDB documents.
// Embed this in your model structs. You must override CollectionName() on your model.
type BaseField struct {
	ID        bson.ObjectID `bson:"_id,omitempty" json:"_id"`
	CreatedAt time.Time     `bson:"createdAt,omitempty" json:"createdAt"`
	UpdatedAt time.Time     `bson:"updatedAt,omitempty" json:"updatedAt"`
}

// CollectionName returns an empty string. Models must override this method.
func (b *BaseField) CollectionName() string {
	return ""
}

// SetID sets the document's ID
func (b *BaseField) SetID(id bson.ObjectID) {
	b.ID = id
}

// DefaultID sets the default value for _id field if it's zero.
func (b *BaseField) DefaultID() {
	if b.ID.IsZero() {
		b.ID = bson.NewObjectID()
	}
}

// DefaultCreatedAt sets the creation timestamp
func (b *BaseField) DefaultCreatedAt() {
	b.CreatedAt = time.Now()
}

// DefaultUpdatedAt sets the update timestamp
func (b *BaseField) DefaultUpdatedAt() {
	b.UpdatedAt = time.Now()
}

// BeforeInsert set default fields before inserting a document
func (b *BaseField) BeforeInsert() {
	b.DefaultID()
	b.DefaultCreatedAt()
	b.DefaultUpdatedAt()
}

// BeforeUpdate set default fields before updating a document
func (b *BaseField) BeforeUpdate() {
	b.DefaultUpdatedAt()
}

// Validate provides a base validation (can be overridden by implementing models)
func (b *BaseField) Validate() error {
	// Add custom validation logic here
	return nil
}
