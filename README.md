# mongokit

Generic, type-safe MongoDB repository for Go. Wraps the official MongoDB driver with a clean API that handles collections, indexes, timestamps, and pagination out of the box.

## Install

```bash
go get github.com/DmitriyHellyeah/mongokit
```

## Quick Start

### Define a model

```go
type User struct {
    mongokit.BaseField `bson:",inline"`
    Name               string `bson:"name"`
    Email              string `bson:"email"`
}

func (*User) CollectionName() string { return "users" }
```

`BaseField` gives you `_id`, `createdAt`, and `updatedAt` fields automatically:

- **`BeforeInsert`** - called on every `InsertOne`/`InsertMany`. Generates `_id` if empty, sets `createdAt` and `updatedAt` to current time.
- **`BeforeUpdate`** - called on update methods when passing a struct (not `bson.M`). Updates `updatedAt` to current time.

> Note: when updating with `bson.M{"$set": ...}` or `bson.M{"$inc": ...}`, timestamps are not updated automatically. Use a struct to get auto-timestamps, or add `updatedAt` to your operator manually.

### Connect and create a repository

```go
db, err := mongokit.Connect(ctx, "mongodb://localhost:27017", "myapp")
if err != nil {
    log.Fatal(err)
}

users, err := mongokit.NewRepository[*User](ctx, db)
if err != nil {
    log.Fatal(err)
}
```

### CRUD

```go
// Insert - ID and timestamps set automatically
user, err := users.InsertOne(ctx, &User{Name: "John", Email: "john@example.com"})

// Find
user, err := users.FindByID(ctx, id)
user, err := users.FindOne(ctx, bson.M{"email": "john@example.com"})
all, err := users.FindDecoded(ctx, bson.M{"status": "active"})

// Update
users.UpdateByID(ctx, id, bson.M{"$set": bson.M{"name": "Jane"}})

// Delete
users.DeleteByID(ctx, id)
```

### Indexes

Define indexes on the model - they are created automatically when the repository is initialized:

```go
func (*User) Indexes() []mongo.IndexModel {
    return mongokit.BuildIndexes(
        mongokit.UniqueIndexes("email"),
        mongokit.NonUniqueIndexes("status", "-createdAt"),
    )
}
```

String shortcuts: `"-"` prefix for descending, comma for composite keys (e.g. `"userId,-createdAt"`).

You can also use standard `mongo.IndexModel` directly for advanced cases (TTL, partial filters, etc.):

```go
func (*Session) Indexes() []mongo.IndexModel {
    return mongokit.BuildIndexes(
        mongokit.NonUniqueIndexes("userId"),
        []mongo.IndexModel{
            {
                Keys:    bson.D{{Key: "expiresAt", Value: 1}},
                Options: options.Index().SetExpireAfterSeconds(0),
            },
        },
    )
}
```

### Pagination

Offset:

```go
items, total, err := users.FindPaginatedWithTotal(ctx, bson.M{}, 1, 20)
```

Cursor-based:

```go
page, err := users.FindCursorPaginated(ctx,
    bson.M{},
    &mongokit.CursorPagination{Limit: 20, Direction: mongokit.CursorNext},
    bson.D{{Key: "_id", Value: 1}},
    func(u *User) (string, error) { return u.ID.Hex(), nil },
)
// page.Items, page.HasNext, page.HasPrev, page.NextCursor, page.PrevCursor
```

### Custom Repository

Extend the built-in repository with your own methods:

```go
type UserRepo struct {
    *mongokit.Repository[*User]
}

func (r *UserRepo) FindActiveByEmail(ctx context.Context, email string) (*User, error) {
    return r.FindOne(ctx, bson.M{"email": email, "status": "active"})
}
```

### Error Handling

```go
// Map mongo "not found" to your domain error
user, err := users.FindByID(ctx, id)
err = mongokit.MapNotFoundErr(err, ErrUserNotFound)
```

Sentinel errors: `ErrNilFilter`, `ErrNilUpdate`, `ErrNilPipeline`, `ErrNilID`, `ErrEmptySlice`, `ErrEmptyCollectionName`, `ErrUnsupportedUpdateType`.

## Recommended Project Structure

```
myapp/
├── cmd/
│   └── server/
│       └── main.go
├── internal/
│   ├── database/
│   │   ├── database.go        # Database struct, Initialize()
│   │   ├── model/
│   │   │   ├── user.go        # model + CollectionName + Indexes
│   │   │   ├── task.go
│   │   │   └── session.go
│   │   └── repository/
│   │       ├── user.go        # custom UserRepo with domain methods
│   │       └── task.go
│   └── service/
│       └── user.go            # business logic
└── go.mod
```

```go
// internal/database/model/user.go
type User struct {
    mongokit.BaseField `bson:",inline"`
    Name   string `bson:"name"`
    Email  string `bson:"email"`
    Status string `bson:"status"`
}

func (*User) CollectionName() string { return "users" }
func (*User) Indexes() []mongo.IndexModel {
    return mongokit.UniqueIndexes("email")
}
```

```go
// internal/database/repository/user.go
type UserRepo struct {
    *mongokit.Repository[*model.User]
}

func NewUserRepo(ctx context.Context, db *mongo.Database) (*UserRepo, error) {
    repo, err := mongokit.NewRepository[*model.User](ctx, db)
    if err != nil {
        return nil, err
    }
    return &UserRepo{repo}, nil
}

func (r *UserRepo) FindActiveByEmail(ctx context.Context, email string) (*model.User, error) {
    return r.FindOne(ctx, bson.M{"email": email, "status": "active"})
}
```

```go
// internal/database/database.go
type Database struct {
    Users *repository.UserRepo
    Tasks *mongokit.Repository[*model.Task]  // no custom methods needed
}

func Initialize(ctx context.Context, uri, dbName string) (*Database, error) {
    db, err := mongokit.Connect(ctx, uri, dbName)
    if err != nil {
        return nil, err
    }

    users, err := repository.NewUserRepo(ctx, db)
    if err != nil {
        return nil, err
    }

    tasks, err := mongokit.NewRepository[*model.Task](ctx, db)
    if err != nil {
        return nil, err
    }

    return &Database{Users: users, Tasks: tasks}, nil
}
```

```go
// cmd/server/main.go
func main() {
    ctx := context.Background()
    db, err := database.Initialize(ctx, os.Getenv("MONGO_URI"), "myapp")
    if err != nil {
        log.Fatal(err)
    }

    user, err := db.Users.FindActiveByEmail(ctx, "john@example.com")
    task, err := db.Tasks.FindByID(ctx, taskID)
}
```

## Examples

See [`examples/`](examples/) for runnable demos:

- [`basic`](examples/basic/) - CRUD, timestamps, MapNotFoundErr
- [`indexes`](examples/indexes/) - all index variants
- [`transactions`](examples/transactions/) - transaction with rollback
- [`pagination`](examples/pagination/) - offset + cursor
- [`custom-repo`](examples/custom-repo/) - extended repository

## License

MIT
