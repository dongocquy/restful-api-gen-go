package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/dongocquy/restful-api-gen-go/generator"
	"github.com/dongocquy/restful-api-gen-go/parser"
	"github.com/dongocquy/restful-api-gen-go/schema"
)

func main() {
	// CLI flags
	mode := flag.String("mode", "from-json", "Generation mode: from-json, from-db, sample, serve")
	jsonPath := flag.String("json", "table.json", "Path to table.json file")
	outputDir := flag.String("output", "./generated", "Output directory for generated code")
	serve := flag.Bool("serve", false, "Generate main.go + build + run server after generation")
	port := flag.String("port", "8080", "Server port (serve mode)")
	packageName := flag.String("package", "api", "Go package name")
	modulePath := flag.String("module", "github.com/yourorg/api-server", "Go module path")
	apiVersion := flag.String("api-version", "v1", "API version prefix")
	framework := flag.String("framework", "gin", "Web framework: gin, chi, echo, fiber")

	// DB connection flags
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

	case "serve":
		// serve = from-db + generate main.go + build + run
		cfg := buildDBConfig(*packageName, *modulePath, *apiVersion, *framework,
			*dbHost, *dbPort, *dbName, *dbUser, *dbPass, *dbSchema)
		generateAndRun(cfg, *jsonPath, *outputDir, *port)

	case "from-db":
		cfg := buildDBConfig(*packageName, *modulePath, *apiVersion, *framework,
			*dbHost, *dbPort, *dbName, *dbUser, *dbPass, *dbSchema)
		generateOutput(cfg, *jsonPath, *outputDir, *port, *serve)

	case "from-json":
		project, err := parser.ParseJSONFile(*jsonPath)
		if err != nil {
			log.Fatalf("Failed to parse %s: %v", *jsonPath, err)
		}
		fmt.Printf("  ✓ Loaded %d tables\n", len(project.Tables))
		for _, t := range project.Tables {
			fmt.Printf("    - %s (%s)\n", t.Name, t.Alias)
		}
		generateOutput(project, *jsonPath, *outputDir, *port, *serve)

	default:
		log.Fatalf("Unknown mode: %s (use: from-json, from-db, sample, serve)", *mode)
	}
}

func buildDBConfig(pkgName, modPath, apiVer, framework, host string, port int, dbName, user, pass, dbSchema string) *schema.ProjectConfig {
	dbCfg := schema.DBConfig{
		Type:     "sqlserver",
		Host:     host,
		Port:     port,
		Database: dbName,
		Schema:   dbSchema,
		User:     user,
		Password: pass,
	}

	fmt.Println("🔌 Connecting to SQL Server...")
	project, err := parser.ParseSQLServer(dbCfg, pkgName, modPath, apiVer, framework)
	if err != nil {
		log.Fatalf("Failed to parse SQL Server: %v", err)
	}
	fmt.Printf("  ✓ Found %d tables\n", len(project.Tables))
	return project
}

func generateOutput(project *schema.ProjectConfig, jsonPath, outputDir, port string, withServe bool) {
	// Save table.json
	if jsonPath != "" {
		fmt.Printf("📄 Saving table.json to %s...\n", jsonPath)
		_ = parser.SaveToJSON(project, jsonPath)
		fmt.Println("  ✓ table.json saved")
	}

	if withServe {
		generateAndRun(project, jsonPath, outputDir, port)
		return
	}

	fmt.Println("\n⚙️  Generating code...")
	if err := generator.Generate(project, outputDir); err != nil {
		log.Fatalf("Generation failed: %v", err)
	}

	pkgDir := filepath.Join(outputDir, project.PackageName)
	fmt.Printf("\n✅ Complete! Output: %s/\n", pkgDir)
	fmt.Println("\nNext steps:")
	fmt.Printf("  cd %s\n", pkgDir)
	fmt.Println("  go mod tidy")
	fmt.Println("  go run .")
}

func generateAndRun(project *schema.ProjectConfig, jsonPath, outputDir, port string) {
	fmt.Println("\n⚙️  Generating code + main.go...")
	if err := generator.GenerateFull(project, outputDir, port); err != nil {
		log.Fatalf("Generation failed: %v", err)
	}

	pkgDir := filepath.Join(outputDir, project.PackageName)
	absDir, _ := filepath.Abs(pkgDir)

	fmt.Println("\n📦 Installing dependencies...")
	cmd := exec.Command("go", "mod", "tidy")
	cmd.Dir = outputDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		log.Fatalf("go mod tidy failed: %v", err)
	}
	fmt.Println("  ✓ Dependencies installed")

	fmt.Println("\n🏗️  Building...")
	cmd = exec.Command("go", "build", "-o", "server", ".")
	cmd.Dir = outputDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		log.Fatalf("Build failed: %v", err)
	}
	fmt.Println("  ✓ Build successful")

	fmt.Printf("\n🚀 Server starting on port %s...\n", port)
	fmt.Printf("   Directory: %s/\n", absDir)
	fmt.Println("   Press Ctrl+C to stop")
	fmt.Println()

	// Run the server
	serverPath := filepath.Join(outputDir, "server")
	serverCmd := exec.Command(serverPath)
	serverCmd.Stdout = os.Stdout
	serverCmd.Stderr = os.Stderr
	serverCmd.Dir = pkgDir

	// Forward interrupt signals
	serverCmd.Stdin = os.Stdin

	if err := serverCmd.Run(); err != nil {
		log.Fatalf("Server exited: %v", err)
	}
}

func generateSample(jsonPath string) {
	fmt.Println("📝 Writing sample table.json...")
	if err := parser.WriteSampleTableJSON(jsonPath); err != nil {
		log.Fatalf("Failed to write sample: %v", err)
	}
	fmt.Printf("  ✓ Sample written to %s\n", jsonPath)
	fmt.Println("\nEdit this file, then run:")
	fmt.Printf("  go run main.go -mode from-json -json %s -output ./generated\n", jsonPath)
	fmt.Println("  # Or serve directly:")
	fmt.Printf("  go run . -mode from-json -json %s -output ./generated -serve\n", jsonPath)
}

func init() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, `RESTful API Generator for Go — Auto-generate REST APIs from SQL Server

Usage:
  %[1]s -mode serve -db-name mydb -db-pass '...'                         # 1 command → API running
  %[1]s -mode serve -db-name mydb -db-pass '...' -port 3000              # Custom port
  %[1]s -mode from-json -json table.json                                 # Gen code from file
  %[1]s -mode from-json -json table.json -serve                          # Gen + build + run
  %[1]s -mode from-db -db-name mydb -db-pass '...'                       # Gen code from DB
  %[1]s -mode sample                                                     # Create sample table.json

Flags:
`, os.Args[0])
		flag.PrintDefaults()
	}
}
