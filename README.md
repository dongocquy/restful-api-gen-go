# restful-api-gen-go 🚀

**Tự động sinh RESTful API + OpenAPI spec từ SQL Server schema hoặc file `table.json`**

Define schema **1 lần** → system tự sinh **toàn bộ** endpoint RESTful + OpenAPI spec. Không cần code từng endpoint.

---

## 🌟 Tính năng

- **Tự động gen** Model → Repository → Service → Handler → Router
- **OpenAPI 3.0.3 spec** — tương thích Swagger UI / Redoc / Postman
- **SQL Server native** — OFFSET/FETCH pagination, SCOPE_IDENTITY, soft delete
- **Gin framework** — handlers ready-to-use
- **3 modes**: từ `table.json`, từ SQL Server trực tiếp, hoặc tạo mẫu

## 📦 Output

| File | Layer | Mô tả |
|------|-------|-------|
| `model.go` | Model | Go struct + `json`/`db` tags |
| `repository.go` | Repository | CRUD: GetAll, GetByID, Create, Update, Delete |
| `service.go` | Service | Business logic + pagination + auto timestamp |
| `handler.go` | Handler | Gin HTTP handlers |
| `router.go` | Router | `SetupRoutes(router, db)` |
| `openapi.yaml` | OpenAPI | OpenAPI 3.0.3 spec đầy đủ paths + schemas |

### Endpoints tự động

```
GET    /api/v1/{table}          # List (page, page_size)
GET    /api/v1/{table}/:id      # Get by ID
POST   /api/v1/{table}          # Create
PUT    /api/v1/{table}/:id      # Update
DELETE /api/v1/{table}/:id      # Delete (soft delete nếu có deleted_at)
```

---

## 🚀 Quick Start

### 1. Clone & build

```bash
git clone https://github.com/dongocquy/restful-api-gen-go.git
cd restful-api-gen-go
go build ./...
```

### 2. Tạo sample `table.json`

```bash
go run main.go -mode sample -json my_schema.json
```

Sửa file `my_schema.json` với schema thật.

### 3. Gen code

```bash
go run main.go -mode from-json -json my_schema.json -output ./generated
```

Kết quả:

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

### 4. Gen trực tiếp từ SQL Server

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

→ Tự động đọc schema → tạo `table.json` → gen code.

---

## 📐 Format `table.json`

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

## 🔧 Yêu cầu

- Go 1.22+
- SQL Server driver (cho `from-db` mode)

---

## 📄 OpenAPI

File `openapi.yaml` được gen tự động, tương thích:

- **Swagger UI** — import trực tiếp
- **Redoc** — `npx redoc-cli bundle openapi.yaml`
- **Postman** — Import → OpenAPI
- **Stoplight** — import spec

### View locally

```bash
# Dùng redoc
npx redoc-cli serve output/openapi.yaml

# Hoặc Swagger UI Docker
docker run -p 8081:8080 -v $(pwd)/output:/spec swaggerapi/swagger-ui
```

---

## 🏗️ Architecture

```
table.json ──► Generator ──► model.go
  (hoặc)                      ├── repository.go
                              ├── service.go
                              ├── handler.go
                              ├── router.go
                              └── openapi.yaml
```

**Pattern:** Go code generator sử dụng `fmt.Sprintf` + string building (không text/template) để sinh code Go sạch, đúng format, compile ngay.

---

## 📝 License

MIT
