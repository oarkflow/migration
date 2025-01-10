package migration

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/oarkflow/migration/pluralize"
)

var (
	upRegex          = regexp.MustCompile(`(?s)-- \+migrate Up(.*?)(-- \+migrate Down|$)`)
	createTableRegex = regexp.MustCompile(`(?s)(?i)CREATE TABLE\s+(?:IF NOT EXISTS\s+)?(\w+)\s*\((.*?)\);`)
	columnRegex      = regexp.MustCompile(`(\w+)\s+([^\s,]+(?:\([^\)]+\))?)(.*)`)
)

func GenerateGoStruct(sql, pkgName, dirName, fileName string) string {
	if pkgName == "" {
		pkgName = "models"
	}
	upSection := extractUpSection(sql)
	if upSection == "" {
		return "No -- +migrate Up section found."
	}
	createTableStatement := extractCreateTable(upSection)
	if createTableStatement == "" {
		return "No CREATE TABLE statement found in -- +migrate Up section."
	}
	structCode := parseCreateTable(createTableStatement+";", pkgName)
	if fileName != "" {
		if dirName != "" {
			if err := os.MkdirAll(dirName, os.ModePerm); err != nil {
				return fmt.Sprintf("Failed to create directory '%s': %v", dirName, err)
			}
		}
		filePath := filepath.Join(dirName, fileName)
		if err := os.WriteFile(filePath, []byte(structCode), 0644); err != nil {
			return fmt.Sprintf("Failed to write to file '%s': %v", filePath, err)
		}
		return fmt.Sprintf("Go struct generated and written to file: %s", filePath)
	}
	return structCode
}

func extractUpSection(sql string) string {
	match := upRegex.FindStringSubmatch(sql)
	if len(match) > 1 {
		return match[1]
	}
	return ""
}

func extractCreateTable(upSection string) string {
	statements := strings.Split(upSection, ";")
	for _, stmt := range statements {
		if strings.Contains(stmt, "CREATE TABLE") {
			return stmt
		}
	}
	return ""
}

func parseCreateTable(createTableStatement, pkgName string) string {
	matches := createTableRegex.FindStringSubmatch(createTableStatement)
	if matches == nil {
		return "Invalid CREATE TABLE statement."
	}

	tableName := matches[1]
	columns := matches[2]
	columnLines := strings.Split(columns, ",")
	var fields []string
	var primaryKey string
	var timeCol bool
	for _, columnLine := range columnLines {
		columnLine = strings.TrimSpace(columnLine)
		colMatches := columnRegex.FindStringSubmatch(columnLine)
		if colMatches == nil {
			continue
		}

		columnName := colMatches[1]
		sqlType := colMatches[2]
		columnModifiers := strings.ToLower(colMatches[3])
		goType, hasTime := mapSQLTypeToGoType(sqlType)
		timeCol = hasTime
		if primaryKey == "" && (strings.Contains(columnModifiers, "primary key") ||
			strings.Contains(sqlType, "serial") || strings.Contains(sqlType, "identity") ||
			strings.Contains(columnModifiers, "unique") && strings.Contains(columnModifiers, "not null")) {
			primaryKey = columnName
		}

		fields = append(fields, fmt.Sprintf("\t%s %s `json:\"%s\"`", toCamelCase(columnName), goType, columnName))
	}
	if primaryKey == "" {
		primaryKey = "id"
	}
	structName := toCamelCase(pluralize.NewClient().Singular(tableName))
	st := fmt.Sprintf("package %s \n\n", pkgName)
	st += `import (
	"time"
	"fmt"
)`
	if timeCol {
	} else {
		st += `import (
	"fmt"
)`
	}
	st += fmt.Sprintf("\n\ntype %s struct {\n%s\n}\n", structName, strings.Join(fields, "\n"))
	// Add table name and ID methods
	st += fmt.Sprintf(`
func (u *%s) TableName() string {
	return "%s"
}

func (u *%s) ID() string {
	return fmt.Sprintf("%%v",u.%s)
}
`, structName, tableName, structName, toCamelCase(primaryKey))

	return st
}

func mapSQLTypeToGoType(sqlType string) (string, bool) {
	sqlType = strings.ToLower(sqlType)
	switch {
	case strings.HasPrefix(sqlType, "int8"), strings.HasPrefix(sqlType, "bigint"), strings.HasPrefix(sqlType, "long"):
		return "int64", false
	case strings.HasPrefix(sqlType, "int"), strings.HasPrefix(sqlType, "serial"), strings.HasPrefix(sqlType, "tinyint"):
		return "int", false
	case strings.HasPrefix(sqlType, "smallint"):
		return "int16", false
	case strings.HasPrefix(sqlType, "mediumint"):
		return "int32", false
	case strings.HasPrefix(sqlType, "decimal"), strings.HasPrefix(sqlType, "numeric"), strings.HasPrefix(sqlType, "money"):
		return "float64", false
	case strings.HasPrefix(sqlType, "float"), strings.HasPrefix(sqlType, "real"), strings.HasPrefix(sqlType, "double"):
		return "float64", false
	case strings.HasPrefix(sqlType, "char"), strings.HasPrefix(sqlType, "varchar"), strings.HasPrefix(sqlType, "text"), strings.HasPrefix(sqlType, "clob"):
		return "string", false
	case strings.HasPrefix(sqlType, "bool"), strings.HasPrefix(sqlType, "boolean"), strings.HasPrefix(sqlType, "bit"):
		return "bool", false
	case strings.HasPrefix(sqlType, "timestamp"), strings.HasPrefix(sqlType, "datetime"), strings.HasPrefix(sqlType, "date"), strings.HasPrefix(sqlType, "time"):
		return "time.Time", true
	case strings.HasPrefix(sqlType, "blob"), strings.HasPrefix(sqlType, "binary"), strings.HasPrefix(sqlType, "varbinary"):
		return "[]byte", false
	default:
		return "any", false
	}
}

func toCamelCase(input string) string {
	if strings.HasSuffix(input, "_id") {
		input = strings.TrimSuffix(input, "_id") + "_ID"
	}
	parts := strings.Split(input, "_")
	for i := range parts {
		parts[i] = strings.Title(parts[i])
	}
	return strings.Join(parts, "")
}
