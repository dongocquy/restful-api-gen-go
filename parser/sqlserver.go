package parser

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"os"
	"strings"

	"github.com/dongocquy/restful-api-gen-go/schema"

	_ "github.com/denisenkom/go-mssqldb"
)

// ParseSQLServer connects to SQL Server and extracts schema into ProjectConfig
func ParseSQLServer(cfg schema.DBConfig, packageName, modulePath, apiVersion, framework string) (*schema.ProjectConfig, error) {
	connStr := cfg.ConnStr
	if connStr == "" {
		connStr = buildConnStr(cfg)
	}

	db, err := sql.Open("sqlserver", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to SQL Server: %w", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping SQL Server: %w", err)
	}

	log.Printf("Connected to SQL Server: %s/%s", cfg.Host, cfg.Database)

	schemaName := cfg.Schema
	if schemaName == "" {
		schemaName = "dbo"
	}

	// Get all user tables
	tables, err := getTables(db, schemaName, cfg.Database)
	if err != nil {
		return nil, fmt.Errorf("failed to get tables: %w", err)
	}

	log.Printf("Found %d tables in schema '%s'", len(tables), schemaName)

	var tableDefs []schema.Table
	for _, tbl := range tables {
		tableDef, err := describeTable(db, schemaName, tbl)
		if err != nil {
			log.Printf("Warning: failed to describe table '%s': %v", tbl, err)
			continue
		}
		tableDefs = append(tableDefs, *tableDef)
	}

	project := &schema.ProjectConfig{
		PackageName: packageName,
		ModulePath:  modulePath,
		APIVersion:  apiVersion,
		Framework:   framework,
		Database:    cfg,
		Tables:      tableDefs,
	}

	return project, nil
}

// ParseJSONFile reads a table.json file into ProjectConfig
func ParseJSONFile(path string) (*schema.ProjectConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read %s: %w", path, err)
	}

	var cfg schema.ProjectConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse %s: %w", path, err)
	}

	return &cfg, nil
}

// SaveToJSON writes ProjectConfig to a JSON file
func SaveToJSON(cfg *schema.ProjectConfig, path string) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}
	return os.WriteFile(path, data, 0644)
}

// --- internal helpers ---

func buildConnStr(cfg schema.DBConfig) string {
	return fmt.Sprintf("sqlserver://%s:%s@%s:%d?database=%s&encrypt=disable",
		cfg.User, url.QueryEscape(cfg.Password), cfg.Host, cfg.Port, cfg.Database)
}

func getTables(db *sql.DB, schemaName, database string) ([]string, error) {
	query := `
		SELECT TABLE_NAME
		FROM INFORMATION_SCHEMA.TABLES
		WHERE TABLE_SCHEMA = @p1
		  AND TABLE_TYPE = 'BASE TABLE'
		ORDER BY TABLE_NAME
	`
	rows, err := db.Query(query, schemaName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		tables = append(tables, name)
	}
	return tables, rows.Err()
}

func describeTable(db *sql.DB, schemaName, tableName string) (*schema.Table, error) {
	table := &schema.Table{
		Name:  tableName,
		Alias: toPascalCase(tableName),
	}

	// Get columns
	colQuery := `
		SELECT
			c.COLUMN_NAME,
			c.DATA_TYPE,
			c.CHARACTER_MAXIMUM_LENGTH,
			c.IS_NULLABLE,
			c.COLUMN_DEFAULT,
			COLUMNPROPERTY(OBJECT_ID(@p1+'.'+@p2), c.COLUMN_NAME, 'IsIdentity') AS IS_IDENTITY,
			COLUMNPROPERTY(OBJECT_ID(@p1+'.'+@p2), c.COLUMN_NAME, 'IsPK') AS IS_PK
		FROM INFORMATION_SCHEMA.COLUMNS c
		WHERE c.TABLE_SCHEMA = @p1 AND c.TABLE_NAME = @p2
		ORDER BY c.ORDINAL_POSITION
	`
	fullTableName := schemaName + "." + tableName
	rows, err := db.Query(colQuery, schemaName, tableName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var colName, dataType, isNullable string
		var maxLength sql.NullInt64
		var defaultVal sql.NullString
		var isIdentity, isPK sql.NullInt64

		if err := rows.Scan(&colName, &dataType, &maxLength, &isNullable, &defaultVal, &isIdentity, &isPK); err != nil {
			return nil, err
		}

		col := schema.Column{
			Name:           colName,
			Type:           dataType,
			GoType:         sqlTypeToGoType(dataType, isNullable),
			Nullable:       isNullable == "YES",
			IsAutoIncrement: isIdentity.Valid && isIdentity.Int64 == 1,
			IsPrimaryKey:   isPK.Valid && isPK.Int64 == 1,
		}

		if maxLength.Valid {
			col.MaxLength = int(maxLength.Int64)
		}
		if defaultVal.Valid {
			col.DefaultValue = defaultVal.String
		}

		table.Columns = append(table.Columns, col)

		if col.IsPrimaryKey {
			table.PrimaryKey = colName
		}
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Detect timestamps
	table.Timestamps = hasColumn(table, "created_at") || hasColumn(table, "updated_at")
	table.SoftDelete = hasColumn(table, "deleted_at")

	// Get indexes
	idxQuery := `
		SELECT
			i.name AS index_name,
			COL_NAME(ic.object_id, ic.column_id) AS column_name,
			i.is_unique,
			i.is_primary_key
		FROM sys.indexes i
		INNER JOIN sys.index_columns ic ON i.object_id = ic.object_id AND i.index_id = ic.index_id
		WHERE i.object_id = OBJECT_ID(@p1)
		  AND i.is_primary_key = 0
		ORDER BY i.name, ic.index_column_id
	`
	idxRows, err := db.Query(idxQuery, fullTableName)
	if err != nil {
		return table, nil // indexes are optional, don't fail
	}
	defer idxRows.Close()

	type idxEntry struct {
		Name   string
		Col    string
		Unique bool
		IsPK   bool
	}
	var idxMap = make(map[string]*idxEntry)
	var idxOrder []string

	for idxRows.Next() {
		var idxName, colName string
		var isUnique, isPK bool
		if err := idxRows.Scan(&idxName, &colName, &isUnique, &isPK); err != nil {
			continue
		}
		if isPK {
			continue
		}
		if _, exists := idxMap[idxName]; !exists {
			idxMap[idxName] = &idxEntry{Name: idxName, Unique: isUnique}
			idxOrder = append(idxOrder, idxName)
		}
		idxMap[idxName].Col = colName
		// collect columns...
		// Simplified: single column index only
		table.Indexes = append(table.Indexes, schema.Index{
			Name:    idxName,
			Columns: []string{colName},
			Unique:  isUnique,
		})
	}

	// Get foreign key relations
	fkQuery := `
		SELECT
			fk.name AS fk_name,
			COL_NAME(fkc.parent_object_id, fkc.parent_column_id) AS fk_column,
			OBJECT_NAME(fkc.referenced_object_id) AS ref_table,
			COL_NAME(fkc.referenced_object_id, fkc.referenced_column_id) AS ref_column
		FROM sys.foreign_keys fk
		INNER JOIN sys.foreign_key_columns fkc ON fk.object_id = fkc.constraint_object_id
		WHERE fk.parent_object_id = OBJECT_ID(@p1)
	`
	fkRows, err := db.Query(fkQuery, fullTableName)
	if err == nil {
		defer fkRows.Close()
		for fkRows.Next() {
			var fkName, fkColumn, refTable, refColumn string
			if err := fkRows.Scan(&fkName, &fkColumn, &refTable, &refColumn); err != nil {
				continue
			}
			table.Relations = append(table.Relations, schema.Relation{
				Type:       "belongs_to",
				Table:      refTable,
				ForeignKey: fkColumn,
				Reference:  refColumn,
			})
		}
	}

	return table, nil
}

// --- helpers ---

func hasColumn(table *schema.Table, name string) bool {
	for _, c := range table.Columns {
		if strings.EqualFold(c.Name, name) {
			return true
		}
	}
	return false
}

func toPascalCase(s string) string {
	parts := strings.Split(s, "_")
	for i, p := range parts {
		if len(p) > 0 {
			parts[i] = strings.ToUpper(p[:1]) + strings.ToLower(p[1:])
		}
	}
	// Also handle single table names like "Users"
	result := strings.Join(parts, "")
	if len(result) > 0 {
		// Handle plural -> singular heuristic (basic)
		if strings.HasSuffix(result, "s") && !strings.HasSuffix(result, "ss") {
			result = strings.TrimSuffix(result, "s")
		}
		// Prefix with _ if starts with digit (invalid Go identifier)
		if result[0] >= '0' && result[0] <= '9' {
			result = "_" + result
		}
	}
	return result
}

func sqlTypeToGoType(sqlType string, nullable string) string {
	sqlType = strings.ToLower(sqlType)

	// NOTE: We always return POINTER types (*string, *int64, *float64, *bool, *time.Time)
	// for all column types that could be NULL. This is because INFORMATION_SCHEMA
	// sometimes reports IS_NULLABLE='NO' when the column actually contains NULL
	// (e.g. after ALTER COLUMN, computed columns, etc.).
	// Using pointers ensures sqlx.StructScan always succeeds even when
	// actual data violates the schema metadata.
	//
	// Uniqueidentifier returns raw "string" because Go's sqlx can scan it
	// even from NULL (it becomes ""), and GUIDs are almost never NULL.
	// Binary/image return "[]byte" which is already nil-safe.
	switch sqlType {
	case "int", "integer":
		return "*int"
	case "bigint":
		return "*int64"
	case "smallint":
		return "*int16"
	case "tinyint":
		return "*int8"
	case "bit":
		return "*bool"
	case "decimal", "numeric", "money", "smallmoney":
		return "*float64"
	case "float", "real":
		return "*float64"
	case "nvarchar", "varchar", "nchar", "char", "text", "ntext":
		return "*string"
	case "datetime", "datetime2", "smalldatetime", "date", "time":
		return "*time.Time"
	case "uniqueidentifier":
		return "*string"
	case "varbinary", "binary", "image":
		return "[]byte"
	default:
		return "*string"
	}
}

// goNullType returns the nullable Go pointer type for any database type.
// Use this in repository scanning to always use sql.NullXxx via pointers,
// preventing scan failures when actual data contains NULL despite
// IS_NULLABLE='NO' in the schema.
func goNullType(sqlType string) string {
	sqlType = strings.ToLower(sqlType)
	switch sqlType {
	case "int", "integer":
		return "*int"
	case "bigint":
		return "*int64"
	case "smallint":
		return "*int16"
	case "tinyint":
		return "*int8"
	case "bit":
		return "*bool"
	case "decimal", "numeric", "money", "smallmoney", "float", "real":
		return "*float64"
	case "nvarchar", "varchar", "nchar", "char", "text", "ntext":
		return "*string"
	case "datetime", "datetime2", "smalldatetime", "date", "time":
		return "*time.Time"
	case "uniqueidentifier":
		return "*string"
	case "varbinary", "binary", "image":
		return "[]byte"
	default:
		return "*string"
	}
}

// WriteSampleTableJSON writes an example table.json
func WriteSampleTableJSON(path string) error {
	sample := &schema.ProjectConfig{
		PackageName: "api",
		ModulePath:  "github.com/yourorg/your-api",
		APIVersion:  "v1",
		Framework:   "gin",
		Database: schema.DBConfig{
			Type: "sqlserver",
		},
		Tables: []schema.Table{
			{
				Name:       "users",
				Alias:      "User",
				PrimaryKey: "id",
				Timestamps: true,
				SoftDelete: false,
				Columns: []schema.Column{
					{Name: "id", Type: "int", GoType: "int64", IsAutoIncrement: true, IsPrimaryKey: true, Description: "Primary key"},
					{Name: "name", Type: "nvarchar", GoType: "string", MaxLength: 100, Description: "Full name"},
					{Name: "email", Type: "nvarchar", GoType: "string", MaxLength: 255, Unique: true, Description: "Email address"},
					{Name: "password_hash", Type: "nvarchar", GoType: "string", MaxLength: 255, Description: "bcrypt hash"},
					{Name: "status", Type: "nvarchar", GoType: "string", MaxLength: 20, DefaultValue: "'active'", Description: "active, inactive, suspended"},
					{Name: "created_at", Type: "datetime", GoType: "time.Time", Description: "Record created"},
					{Name: "updated_at", Type: "datetime", GoType: "*time.Time", Nullable: true, Description: "Record updated"},
				},
				Indexes: []schema.Index{
					{Name: "idx_users_email", Columns: []string{"email"}, Unique: true},
					{Name: "idx_users_status", Columns: []string{"status"}},
				},
				Relations: []schema.Relation{
					{Type: "has_many", Table: "orders", ForeignKey: "user_id", Reference: "id"},
				},
			},
			{
				Name:       "orders",
				Alias:      "Order",
				PrimaryKey: "id",
				Timestamps: true,
				Columns: []schema.Column{
					{Name: "id", Type: "int", GoType: "int64", IsAutoIncrement: true, IsPrimaryKey: true},
					{Name: "user_id", Type: "int", GoType: "int64", Description: "FK to users"},
					{Name: "total_amount", Type: "decimal", GoType: "float64", Description: "Order total"},
					{Name: "status", Type: "nvarchar", GoType: "string", MaxLength: 20, DefaultValue: "'pending'"},
					{Name: "created_at", Type: "datetime", GoType: "time.Time"},
					{Name: "updated_at", Type: "datetime", GoType: "*time.Time", Nullable: true},
				},
				Indexes: []schema.Index{
					{Name: "idx_orders_user", Columns: []string{"user_id"}},
				},
			},
		},
	}

	data, _ := json.MarshalIndent(sample, "", "  ")
	return os.WriteFile(path, data, 0644)
}
