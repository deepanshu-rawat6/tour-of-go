# Data Access Object (DAO) & Repository Pattern

As Go applications grow, direct database calls within business logic become a liability. The DAO and Repository patterns decouple your domain logic from persistence details, making your code easier to test and more resilient to infrastructure changes.

---

## 🏗️ The Three-Layer Architecture

1.  **Domain (Business Logic)**: Defines the "what" (entities and interfaces).
2.  **Repository (Interface Implementation)**: Defines the "how" (SQL, NoSQL, In-Memory).
3.  **DAO (Data Access Object)**: Maps raw database rows into domain entities.

---

## 🛠️ The Repository Pattern: Defining the Boundary

Start by defining an interface in your domain layer.

```go
// domain/user.go
type User struct {
    ID   string
    Name string
}

type UserRepository interface {
    GetByID(ctx context.Context, id string) (*User, error)
    Save(ctx context.Context, user *User) error
}
```

### Why a Repository?
It abstracts the data source. Your service layer doesn't care if `GetByID` calls MySQL, Redis, or a Mock during testing.

---

## 🧱 The DAO (Data Access Object): Handling DB Specifics

DAOs are the low-level implementation. They handle the "quirks" of the specific database (e.g., table names, SQL syntax, BSON tags).

```go
// persistence/mysql/user_dao.go
type userDAO struct {
    ID   int64  `db:"id"`
    Name string `db:"full_name"`
}

func (r *mysqlRepo) GetByID(ctx context.Context, id string) (*User, error) {
    var raw userDAO
    err := r.db.GetContext(ctx, &raw, "SELECT id, full_name FROM users WHERE id = ?", id)
    if err != nil { return nil, err }
    
    // Map DAO to Domain Entity
    return &User{ID: strconv.FormatInt(raw.ID, 10), Name: raw.Name}, nil
}
```

---

## 📉 Why Mapping Matters (DAO vs. Domain)

*   **Database Types**: MySQL might use `int64` for an ID, while your domain uses `string` for portability.
*   **Encapsulation**: Don't leak `db` or `json` tags from your database into your business logic.
*   **Flexibility**: You can change your database schema (e.g., renaming a column) by only updating the DAO mapping, leaving your business logic untouched.

---

## 🚀 Key Benefits for Senior Developers
*   **Testability**: You can easily swap a `MySQLRepository` with a `MockRepository` in unit tests.
*   **Swap-ability**: Need to move from MySQL to MongoDB? Only implement a new `mongoRepository` that satisfies the `UserRepository` interface.
*   **Clean Architecture**: Keeps your domain logic "pure" and free of infrastructure dependencies.

---

## 🛠️ Comparison: DAO vs. Repository
| Feature | DAO | Repository |
| :--- | :--- | :--- |
| **Scope** | One table or collection. | One aggregate (e.g., Order + OrderItems). |
| **Abstraction** | Low-level (SQL queries). | High-level (Get, Save, Delete). |
| **Return Value** | DB-specific row/struct. | Domain Entity. |
