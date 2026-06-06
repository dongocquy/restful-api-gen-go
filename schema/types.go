package schema

import "encoding/json"

// ProjectConfig defines the overall project metadata
type ProjectConfig struct {
	PackageName string `json:"package_name"`       // e.g. "api"
	ModulePath  string `json:"module_path"`         // e.g. "github.com/user/project"
	APIVersion  string `json:"api_version"`         // e.g. "v1"
	OutputDir   string `json:"output_dir,omitempty"` // where generated code goes
	Framework   string `json:"framework,omitempty"`  // gin, chi, echo, fiber
	Database    DBConfig `json:"database"`
	Tables      []Table `json:"tables"`
}

// DBConfig holds database connection info (used by parser)
type DBConfig struct {
	Type     string `json:"type"`               // "sqlserver", "postgres", "mysql"
	Host     string `json:"host,omitempty"`
	Port     int    `json:"port,omitempty"`
	Database string `json:"database,omitempty"`
	Schema   string `json:"schema,omitempty"`   // SQL Server schema (default: dbo)
	User     string `json:"user,omitempty"`
	Password string `json:"password,omitempty"`
	ConnStr  string `json:"conn_str,omitempty"` // alternative: raw connection string
}

// Table describes a database table for code generation
type Table struct {
	Name       string   `json:"name"`                 // SQL table name (e.g. "users")
	Alias      string   `json:"alias"`                // Go struct name (e.g. "User")
	PrimaryKey string   `json:"primary_key"`           // primary key column name
	Columns    []Column `json:"columns"`
	Indexes    []Index  `json:"indexes,omitempty"`
	Relations  []Relation `json:"relations,omitempty"`
	SoftDelete bool     `json:"soft_delete,omitempty"` // has deleted_at column?
	Timestamps bool     `json:"timestamps,omitempty"`  // has created_at/updated_at?
}

// Column describes a single column
type Column struct {
	Name           string `json:"name"`
	Type           string `json:"type"`               // SQL type: int, nvarchar, datetime, etc.
	GoType         string `json:"go_type"`            // Go type: int64, string, time.Time, float64, bool
	MaxLength      int    `json:"max_length,omitempty"`
	Nullable       bool   `json:"nullable,omitempty"`
	IsAutoIncrement bool  `json:"is_auto_increment,omitempty"`
	IsPrimaryKey   bool   `json:"is_primary_key,omitempty"`
	DefaultValue   string `json:"default_value,omitempty"`
	Unique         bool   `json:"unique,omitempty"`
	Description    string `json:"description,omitempty"`
	Enum           []string `json:"enum,omitempty"`    // for enum-like columns
	JSONOmitEmpty  bool   `json:"json_omit_empty,omitempty"`
}

// Index describes a database index
type Index struct {
	Name    string   `json:"name"`
	Columns []string `json:"columns"`
	Unique  bool     `json:"unique,omitempty"`
}

// Relation describes a foreign-key relationship
type Relation struct {
	Type       string `json:"type"`                 // "belongs_to", "has_many", "has_one", "many_to_many"
	Table      string `json:"table"`                // related table name
	ForeignKey string `json:"foreign_key"`          // FK column in this table (or junction for m2m)
	Reference  string `json:"reference"`            // ref column in related table (usually PK)
	Junction   string `json:"junction,omitempty"`   // for many_to_many: junction table name
	JunctionFK string `json:"junction_fk,omitempty"`
	JunctionRef string `json:"junction_ref,omitempty"`
}

// FromJSON parses JSON bytes into ProjectConfig
func (c *ProjectConfig) FromJSON(data []byte) error {
	return json.Unmarshal(data, c)
}

// ToJSON serializes ProjectConfig to JSON
func (c *ProjectConfig) ToJSON() ([]byte, error) {
	return json.MarshalIndent(c, "", "  ")
}

// HTTPMethod constants
type HTTPMethod string

const (
	GET    HTTPMethod = "GET"
	POST   HTTPMethod = "POST"
	PUT    HTTPMethod = "PUT"
	DELETE HTTPMethod = "DELETE"
	PATCH  HTTPMethod = "PATCH"
)

// Endpoint describes a generated REST endpoint
type Endpoint struct {
	Method      HTTPMethod `json:"method"`
	Path        string     `json:"path"`
	Description string     `json:"description"`
	HandlerName string     `json:"handler_name"`
}
