package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/dongocquy/restful-api-gen-go/generator"
	"github.com/dongocquy/restful-api-gen-go/parser"
	"github.com/dongocquy/restful-api-gen-go/schema"
)

func main() {
	// CLI flags
	mode := flag.String("mode", "from-json", "Generation mode: from-json, from-db, sample")
	jsonPath := flag.String("json", "table.json", "Path to table.json file (from-json mode)")
	outputDir := flag.String("output", "./output", "Output directory for generated code")
	packageName := flag.String("package", "api", "Go package name (from-db mode)")
	modulePath := flag.String("module", "github.com/yourorg/your-api", "Go module path (from-db mode)")
	apiVersion := flag.String("api-version", "v1", "API version prefix")
	framework := flag.String("framework", "gin", "Web framework: gin, chi, echo, fiber")

	// DB connection flags (from-db mode)
	dbHost := flag.String("db-host", "localhost", "Database host")
	dbPort := flag.Int("db-port", 1433, "Database port")
	dbName := flag.String("db-name", "", "Database name")
	dbUser := flag.String("db-user", "sa", "Database user")
	dbPass := flag.String("db-pass", "", "Database password")
	dbSchema := flag.String("db-schema", "dbo", "Database schema")

	flag.Parse()

	switch *mode {
	case "sample":
		generateSample(*jsonPath)
	case "from-db":
		generateFromDB(*jsonPath, *outputDir, *packageName, *modulePath, *apiVersion, *framework,
			*dbHost, *dbPort, *dbName, *dbUser, *dbPass, *dbSchema)
	case "from-json":
		generateFromJSON(*jsonPath, *outputDir)
	default:
		log.Fatalf("Unknown mode: %s (use: from-json, from-db, sample)", *mode)
	}
}

func generateSample(jsonPath string) {
	fmt.Println("📝 Writing sample table.json...")
	if err := parser.WriteSampleTableJSON(jsonPath); err != nil {
		log.Fatalf("Failed to write sample: %v", err)
	}
	fmt.Printf("  ✓ Sample written to %s\n", jsonPath)
	fmt.Println("\nEdit this file, then run:")
	fmt.Printf("  go run main.go -mode from-json -json %s -output ./output\n", jsonPath)
}

func generateFromDB(jsonPath, outputDir, pkgName, modPath, apiVer, framework,
	host string, port int, dbName, user, pass, dbSchema string) {

	fmt.Println("🔌 Connecting to SQL Server...")
	dbCfg := schema.DBConfig{
		Type:     "sqlserver",
		Host:     host,
		Port:     port,
		Database: dbName,
		Schema:   dbSchema,
		User:     user,
		Password: pass,
	}

	project, err := parser.ParseSQLServer(dbCfg, pkgName, modPath, apiVer, framework)
	if err != nil {
		log.Fatalf("Failed to parse SQL Server: %v", err)
	}

	// Save table.json
	fmt.Printf("📄 Saving table.json to %s...\n", jsonPath)
	if err := parser.SaveToJSON(project, jsonPath); err != nil {
		log.Fatalf("Failed to save table.json: %v", err)
	}
	fmt.Println("  ✓ table.json saved")

	// Generate code
	generateOutput(project, outputDir)
}

func generateFromJSON(jsonPath, outputDir string) {
	fmt.Printf("📖 Reading %s...\n", jsonPath)

	project, err := parser.ParseJSONFile(jsonPath)
	if err != nil {
		log.Fatalf("Failed to parse %s: %v", jsonPath, err)
	}

	fmt.Printf("  ✓ Loaded %d tables\n", len(project.Tables))
	for _, t := range project.Tables {
		fmt.Printf("    - %s (%s) → %s\n", t.Name, t.Alias, outputDir)
	}

	generateOutput(project, outputDir)
}

func generateOutput(project *schema.ProjectConfig, outputDir string) {
	fmt.Println("\n⚙️  Generating code...")
	if err := generator.Generate(project, outputDir); err != nil {
		log.Fatalf("Generation failed: %v", err)
	}
	fmt.Println("\n✅ Code generation complete!")
	fmt.Printf("   Output: %s/\n", outputDir)
	fmt.Printf("   Package: %s\n", project.PackageName)
	fmt.Printf("\nNext steps:\n")
	fmt.Printf("  1. cd %s/%s\n", outputDir, project.PackageName)
	fmt.Printf("  2. go mod tidy\n")
	fmt.Printf("  3. Implement the database connection in main.go\n")
	fmt.Printf("  4. Run: go run main.go\n")
}

func init() {
	// Create output directory if it doesn't exist
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, `RESTful API Generator for Go
Generates full CRUD REST APIs from SQL Server schema or table.json

Usage:
  %[1]s -mode sample                      # Generate sample table.json
  %[1]s -mode from-json -json table.json  # Generate from table.json
  %[1]s -mode from-db -db-name mydb ...   # Generate from SQL Server

Flags:
`, os.Args[0])
		flag.PrintDefaults()
	}
}
