# restful-api-gen-go 🚀

**Auto-generate RESTful APIs + OpenAPI spec from SQL Server schema or a `table.json` file.**

Define your schema **once** — the system generates every REST endpoint and an OpenAPI 3.0.3 spec automatically. No hand-coding individual endpoints.

---

## 🌟 Features

- **Full-stack code generation** — Model → Repository → Service → Handler → Router
- **OpenAPI 3.0.3 spec** — compatible with Swagger UI, Redoc, Postman
- **SQL Server native** — OFFSET/FETCH pagination, SCOPE_IDENTITY, soft delete support
- **Gin framework** — ready-to-use HTTP handlers
- **3 modes**: from `table.json`, direct from SQL Server, or generate a sample

## 📦 Output per schema

| File | Layer | Description |
|------|-------|-------------|
| `model.go` | Model | Go struct with `json`/`db` tags |
| `repository.go` | Repository | CRUD: GetAll, GetByID, Create, Update, Delete |
| `service.go` | Service | Business logic + pagination + auto timestamps |
| `handler.go` | Handler | Gin HTTP handlers |
| `router.go` | Router | `SetupRoutes(router, db)` |
| `openapi.yaml` | OpenAPI | Full OpenAPI 3.0.3 spec |

### Generated endpoints

```
GET    /api/v1/{table}          # List (page, page_size)
GET    /api/v1/{table}/:id      # Get by ID
POST   /api/v1/{table}          # Create
PUT    /api/v1/{table}/:id      # Update
DELETE /api/v1/{table}/:id      # Delete (soft-delete if deleted_at column exists)
```

---

## 🚀 Quick Start

### 1. Clone & build

```bash
git clone https://github.com/dongocquy/restful-api-gen-go.git
cd restful-api-gen-go
go build ./...
```

### 2. Generate a sample `table.json`

```bash
go run main.go -mode sample -json my_schema.json
```

Edit `my_schema.json` with your actual schema.

### 3. Generate code from the JSON file

```bash
go run main.go -mode from-json -json my_schema.json -output ./generated
```

Result:

```
./generated/
├── api/
│   ├── model.go
│   ├── repository.go
│   ├── service.go
│   ├── handler.go
│   └── router.go
└── openapi.yaml
```

### 4. Generate directly from a live SQL Server

```bash
go run main.go -mode from-db \
  -db-host 192.168.1.100 \
  -db-port 1433 \
  -db-name MyDatabase \
  -db-user sa \
  -db-pass 'password' \
  -json table.json \
  -output ./generated
```

→ Automatically reads the schema, produces `table.json`, then generates all code.

---

## 📐 `table.json` Format

```json
{
  "package_name": "api",
  "module_path": "github.com/yourorg/your-api",
  "api_version": "v1",
  "framework": "gin",
  "database": {
    "type": "sqlserver",
    "host": "localhost",
    "port": 1433
  },
  "tables": [
    {
      "name": "users",
      "alias": "User",
      "primary_key": "id",
      "timestamps": true,
      "columns": [
        {"name": "id", "type": "int", "go_type": "int64",
         "is_auto_increment": true, "is_primary_key": true,
         "description": "Primary key"},
        {"name": "name", "type": "nvarchar", "go_type": "string",
         "max_length": 100, "description": "Full name"},
        {"name": "email", "type": "nvarchar", "go_type": "string",
         "max_length": 255, "unique": true, "description": "Email"},
        {"name": "created_at", "type": "datetime", "go_type": "time.Time",
         "description": "Record created"},
        {"name": "updated_at", "type": "datetime", "go_type": "*time.Time",
         "nullable": true, "description": "Record updated"}
      ],
      "indexes": [
        {"name": "idx_email", "columns": ["email"], "unique": true}
      ],
      "relations": [
        {"type": "has_many", "table": "orders",
         "foreign_key": "user_id", "reference": "id"}
      ]
    }
  ]
}
```

---

## 🔧 Requirements

- Go 1.22+
- SQL Server driver (only needed for `from-db` mode)

---

## 📄 OpenAPI

The `openapi.yaml` file is generated automatically and works with:

- **Swagger UI** — import directly
- **Redoc** — `npx redoc-cli bundle openapi.yaml`
- **Postman** — Import → OpenAPI
- **Stoplight** — import the spec

### View locally

```bash
# Using redoc
npx redoc-cli serve output/openapi.yaml

# Or via Swagger UI in Docker
docker run -p 8081:8080 -v $(pwd)/output:/spec swaggerapi/swagger-ui
```

---

## 🏗️ Architecture

```
table.json ──► Generator ──► model.go
  (or)                        ├── repository.go
                              ├── service.go
                              ├── handler.go
                              ├── router.go
                              └── openapi.yaml
```

**Pattern:** A Go code generator that uses `fmt.Sprintf` + string building (not text/template) to emit clean, properly formatted Go code that compiles immediately.

---

## 📝 License

MIT
