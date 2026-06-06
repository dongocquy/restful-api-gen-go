package generator

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/dongocquy/restful-api-gen-go/schema"
)

// GenerateOpenAPI creates openapi.yaml from the project config
func GenerateOpenAPI(cfg *schema.ProjectConfig, outputDir string) error {
	var b strings.Builder

	apiVersion := cfg.APIVersion
	if apiVersion == "" {
		apiVersion = "v1"
	}

	title := "RESTful API"
	if cfg.ModulePath != "" {
		title = cfg.ModulePath
	}

	b.WriteString(fmt.Sprintf(`openapi: "3.0.3"
info:
  title: "%s"
  description: "Auto-generated RESTful API from table.json"
  version: "%s"
servers:
  - url: http://localhost:8080/api/%s
    description: "Local development"

paths:
`, title, apiVersion, apiVersion))

	for _, t := range cfg.Tables {
		pkType := resolvePKType(t)
		pkFormat := pkFormatForOpenAPI(pkType)

		// GET /{table} — List
		b.WriteString(fmt.Sprintf(`  /%s:
    get:
      operationId: list%s
      summary: "List all %s records"
      tags:
        - %s
      parameters:
        - name: page
          in: query
          description: "Page number"
          schema:
            type: integer
            default: 1
        - name: page_size
          in: query
          description: "Items per page (max 100)"
          schema:
            type: integer
            default: 20
      responses:
        "200":
          description: "Paginated list of %s"
          content:
            application/json:
              schema:
                type: object
                properties:
                  data:
                    type: array
                    items:
                      $ref: "#/components/schemas/%s"
                  total:
                    type: integer
                  page:
                    type: integer
                  page_size:
                    type: integer
                  total_pages:
                    type: integer

`, t.Name, t.Alias, t.Name, t.Name, t.Name, t.Alias))

		// POST /{table} — Create
		b.WriteString(fmt.Sprintf(`    post:
      operationId: create%s
      summary: "Create a new %s record"
      tags:
        - %s
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: "#/components/schemas/%s"
      responses:
        "201":
          description: "Created %s"
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/%s"
        "400":
          description: "Bad request"

`, t.Alias, t.Name, t.Name, t.Alias, t.Alias, t.Alias))

		// /{table}/{id} — Get, Update, Delete
		b.WriteString(fmt.Sprintf(`  /%s/{id}:
    get:
      operationId: get%s
      summary: "Get a single %s by ID"
      tags:
        - %s
      parameters:
        - name: id
          in: path
          required: true
          description: "%s ID"
          schema:
            type: %s
            %s
      responses:
        "200":
          description: "The requested %s"
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/%s"
        "404":
          description: "%s not found"

`, t.Name, t.Alias, t.Alias, t.Name, t.Alias, pkType, pkFormat, t.Alias, t.Alias, t.Alias))

		// PUT /{table}/{id} — Update
		b.WriteString(fmt.Sprintf(`    put:
      operationId: update%s
      summary: "Update an existing %s record"
      tags:
        - %s
      parameters:
        - name: id
          in: path
          required: true
          description: "%s ID"
          schema:
            type: %s
            %s
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: "#/components/schemas/%s"
      responses:
        "200":
          description: "Updated %s"
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/%s"
        "400":
          description: "Bad request"
        "404":
          description: "%s not found"

`, t.Alias, t.Alias, t.Name, t.Alias, pkType, pkFormat, t.Alias, t.Alias, t.Alias, t.Alias))

		// DELETE /{table}/{id} — Delete
		b.WriteString(fmt.Sprintf(`    delete:
      operationId: delete%s
      summary: "Delete a %s record"
      tags:
        - %s
      parameters:
        - name: id
          in: path
          required: true
          description: "%s ID"
          schema:
            type: %s
            %s
      responses:
        "204":
          description: "%s deleted successfully"
        "404":
          description: "%s not found"

`, t.Alias, t.Alias, t.Name, t.Alias, pkType, pkFormat, t.Alias, t.Alias))
	}

	// --- Components / Schemas ---
	b.WriteString(`components:
  schemas:
`)

	for _, t := range cfg.Tables {
		b.WriteString(fmt.Sprintf(`    %s:
      type: object
      description: "Represents the %s table"
      properties:
`, t.Alias, t.Name))

		for _, c := range t.Columns {
			oaType, oaFormat := goTypeToOpenAPI(c.GoType)
			nullable := c.Nullable
			desc := c.Description
			if desc == "" {
				desc = c.Name
			}
			b.WriteString(fmt.Sprintf(`        %s:
          type: %s
`, c.Name, oaType))
			if oaFormat != "" {
				b.WriteString(fmt.Sprintf(`          format: %s
`, oaFormat))
			}
			if nullable {
				b.WriteString(`          nullable: true
`)
			}
			b.WriteString(fmt.Sprintf(`          description: "%s"
`, desc))
			if c.MaxLength > 0 {
				b.WriteString(fmt.Sprintf(`          maxLength: %d
`, c.MaxLength))
			}
			if c.IsPrimaryKey || c.IsAutoIncrement {
				b.WriteString(`          readOnly: true
`)
			}
		}

		b.WriteString("\n")
	}

	// Write file
	outputPath := filepath.Join(outputDir, "openapi.yaml")
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}
	if err := os.WriteFile(outputPath, []byte(b.String()), 0644); err != nil {
		return fmt.Errorf("write openapi.yaml: %w", err)
	}

	fmt.Printf("  ✓ Generated: %s\n", outputPath)
	return nil
}

// resolvePKType returns the OpenAPI type for the primary key
func resolvePKType(t schema.Table) string {
	for _, c := range t.Columns {
		if c.IsPrimaryKey {
			return oaTypeForPK(c.GoType)
		}
	}
	return "integer"
}

func oaTypeForPK(goType string) string {
	switch goType {
	case "int", "int64", "int32", "int16", "int8", "uint", "uint64":
		return "integer"
	case "string":
		return "string"
	default:
		return "integer"
	}
}

func pkFormatForOpenAPI(typ string) string {
	switch typ {
	case "integer":
		return "format: int64"
	case "string":
		return ""
	default:
		return ""
	}
}

// goTypeToOpenAPI converts Go types to OpenAPI types
func goTypeToOpenAPI(goType string) (oaType string, oaFormat string) {
	switch goType {
	case "int", "int64":
		return "integer", "int64"
	case "int32":
		return "integer", "int32"
	case "int16":
		return "integer", "int16"
	case "int8":
		return "integer", "int8"
	case "float64", "float32":
		return "number", "double"
	case "bool":
		return "boolean", ""
	case "string":
		return "string", ""
	case "*string":
		return "string", ""
	case "time.Time", "*time.Time":
		return "string", "date-time"
	case "[]byte":
		return "string", "byte"
	default:
		return "string", ""
	}
}
