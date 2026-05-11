package mongokit

import (
	"errors"
	"reflect"
	"strings"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// MapNotFoundErr replaces mongo.ErrNoDocuments with a domain-specific error.
// Returns the original error unchanged if it's not a "not found" error.
func MapNotFoundErr(err, notFound error) error {
	if errors.Is(err, mongo.ErrNoDocuments) {
		return notFound
	}
	return err
}

func isStructOrPtrToStruct(v any) bool {
	t := reflect.TypeOf(v)
	if t == nil {
		return false
	}
	return t.Kind() == reflect.Struct || (t.Kind() == reflect.Ptr && t.Elem().Kind() == reflect.Struct)
}

// prepareUpdate wraps struct payloads in $set and calls BeforeUpdate if available.
// Mongo operators (bson.M with $ keys) pass through unchanged.
// Returns (prepared update, error) - error if unsupported type.
func prepareUpdate(update any) (any, error) {
	if update == nil {
		return nil, ErrNilUpdate
	}
	if isMongoOperator(update) {
		return update, nil
	}
	if isStructOrPtrToStruct(update) {
		if hook, ok := update.(Document); ok {
			hook.BeforeUpdate()
		}
		return bson.M{"$set": update}, nil
	}
	return nil, ErrUnsupportedUpdateType
}

func isMongoOperator(v any) bool {
	m, ok := v.(bson.M)
	if !ok {
		return false
	}
	for k := range m {
		if len(k) > 0 && k[0] == '$' {
			return true
		}
	}
	return false
}

// SplitSortField parses a field string into a key and sort direction.
// A leading "-" means descending (-1), "+" or no prefix means ascending (1).
//
//	"email"      -> ("email", 1)
//	"+email"     -> ("email", 1)
//	"-createdAt" -> ("createdAt", -1)
func SplitSortField(field string) (key string, sort int32) {
	key = strings.TrimSpace(field)
	sort = 1

	if key == "" {
		return
	}

	switch key[0] {
	case '+':
		key = key[1:]
		sort = 1
	case '-':
		key = key[1:]
		sort = -1
	}

	return
}

// UniqueIndexes creates unique index models from string definitions.
// Supports composite indexes via comma separation and "-" prefix for descending.
//
//	mongokit.UniqueIndexes("email", "userId,-createdAt")
func UniqueIndexes(fields ...string) []mongo.IndexModel {
	var models []mongo.IndexModel
	for _, index := range fields {
		var keys bson.D
		for _, field := range strings.Split(index, ",") {
			key, sort := SplitSortField(field)
			keys = append(keys, bson.E{Key: key, Value: sort})
		}
		models = append(models, mongo.IndexModel{
			Keys:    keys,
			Options: options.Index().SetUnique(true),
		})
	}
	return models
}

// NonUniqueIndexes creates non-unique index models from string definitions.
// Supports composite indexes via comma separation and "-" prefix for descending.
//
//	mongokit.NonUniqueIndexes("status", "-score,userId")
func NonUniqueIndexes(fields ...string) []mongo.IndexModel {
	var models []mongo.IndexModel
	for _, index := range fields {
		var keys bson.D
		for _, field := range strings.Split(index, ",") {
			key, sort := SplitSortField(field)
			keys = append(keys, bson.E{Key: key, Value: sort})
		}
		models = append(models, mongo.IndexModel{
			Keys: keys,
		})
	}
	return models
}

// BuildIndexes merges multiple []mongo.IndexModel slices into one.
// Use with UniqueIndexes, NonUniqueIndexes, and manual []mongo.IndexModel.
//
//	mongokit.BuildIndexes(
//	    mongokit.UniqueIndexes("email"),
//	    mongokit.NonUniqueIndexes("status", "-createdAt"),
//	    []mongo.IndexModel{{Keys: bson.D{{Key: "expiresAt", Value: 1}}, Options: options.Index().SetExpireAfterSeconds(0)}},
//	)
func BuildIndexes(groups ...[]mongo.IndexModel) []mongo.IndexModel {
	var models []mongo.IndexModel
	for _, group := range groups {
		models = append(models, group...)
	}
	return models
}
