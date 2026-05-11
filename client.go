package mongokit

import (
	"context"
	"strings"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver/connstring"
)

// Connect connects to MongoDB by URI, pings the server, and returns the database.
// Automatically enables Stable API (ServerAPIVersion1) for SRV URIs.
func Connect(ctx context.Context, uri, database string) (*mongo.Database, error) {
	opts := options.Client().ApplyURI(uri)

	// Configure the MongoDB API version.
	if strings.Contains(uri, connstring.SchemeMongoDBSRV) {
		opts.SetServerAPIOptions(options.ServerAPI(options.ServerAPIVersion1))
	}

	return ConnectWithOptions(ctx, database, opts)
}

// ConnectWithOptions connects to MongoDB with custom client options, pings the server, and returns the database.
func ConnectWithOptions(ctx context.Context, database string, opts ...*options.ClientOptions) (*mongo.Database, error) {
	client, err := mongo.Connect(opts...)
	if err != nil {
		return nil, err
	}

	if err = client.Ping(ctx, nil); err != nil {
		_ = client.Disconnect(context.Background()) // best-effort cleanup; ping error is more important
		return nil, err
	}

	return client.Database(database), nil
}
