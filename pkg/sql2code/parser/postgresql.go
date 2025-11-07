package parser

import (
	"fmt"
	"strings"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// GetPostgresqlTableInfo get table info from postgres
func GetPostgresqlTableInfo(dsn string, tableName string) (PGFields, error) {
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("GetPostgresqlTableInfo error: %v", err)
	}
	defer closeDB(db)

	return getPostgresqlTableFields(db, tableName)
}

// ConvertToSQLByPgFields convert to mysql table ddl
func ConvertToSQLByPgFields(tableName string, fields PGFields) (string, map[string]string) {
	fieldStr := ""
	pgTypeMap := make(map[string]string) // name:type
	if len(fields) == 0 {
		return "", pgTypeMap
	}

	for _, field := range fields {
		pgTypeMap[field.Name] = getType(field)
		sqlType := field.getMysqlType()
		notnullStr := "not null"
		if !field.Notnull {
			notnullStr = "null"
		}
		comment := strings.ReplaceAll(field.Comment, "'", "\\'")
		fieldStr += fmt.Sprintf("    `%s` %s %s comment '%s',\n", field.Name, sqlType, notnullStr, comment)
	}

	primaryField := fields.getPrimaryField()
	if primaryField != nil {
		fieldStr += fmt.Sprintf("    PRIMARY KEY (`%s`)\n", primaryField.Name)
	} else {
		fieldStr = strings.TrimSuffix(fieldStr, ",\n")
	}
	sqlStr := fmt.Sprintf("CREATE TABLE `%s` (\n%s\n);", tableName, fieldStr)
	return sqlStr, pgTypeMap
}

// PGField postgresql field
type PGField struct {
	Name         string `gorm:"column:name;" json:"name"`
	Type         string `gorm:"column:type;" json:"type"`
	Comment      string `gorm:"column:comment;" json:"comment"`
	Length       int    `gorm:"column:length;" json:"length"`
	Lengthvar    int    `gorm:"column:lengthvar;" json:"lengthvar"`
	Notnull      bool   `gorm:"column:notnull;" json:"notnull"`
	IsPrimaryKey bool   `gorm:"column:is_primary_key;" json:"is_primary_key"`
}

// nolint
func (field *PGField) getMysqlType() string {
	switch field.Type {
	case "smallint", "integer", "smallserial", "serial", "int2", "int4":
		return "int"
	case "bigint", "bigserial", "int8":
		return "bigint"
	case "real", "float4":
		return "float"
	case "double precision", "float8":
		return "double"
	case "decimal", "numeric", "money":
		return "decimal(10, 2)"
	case "character", "character varying", "varchar", "char", "bpchar":
		if field.Lengthvar > 4 {
			return fmt.Sprintf("varchar(%d)", field.Lengthvar-4)
		} else {
			return "varchar(100)"
		}
	case "text":
		return "text"
	case "timestamp", "timestamptz":
		return "timestamp"
	case "date":
		return "date"
	case "time": //nolint
		return "time" //nolint
	case "interval":
		return "year"
	case "json", "jsonb":
		return "json"
	case "boolean", "bool":
		return "bit(1)"
	case "bit":
		return "bit"
	}

	// unknown type convert to varchar
	field.Type = "varchar(100)"

	return field.Type
}

type PGFields []*PGField

func (fields PGFields) getPrimaryField() *PGField {
	var f *PGField
	for _, field := range fields {
		if field.IsPrimaryKey || field.Name == "id" {
			f = field
			return f
		}
	}
	return f
}

func getPostgresqlTableFields(db *gorm.DB, tableName string) (PGFields, error) {
	query := fmt.Sprintf(`SELECT
    a.attname AS name,
    t.typname AS type,
    a.attlen AS length,
    a.atttypmod AS lengthvar,
    a.attnotnull AS notnull,
    b.description AS comment,
    CASE
        WHEN pk.constraint_type = 'PRIMARY KEY' THEN true
        ELSE false
        END AS is_primary_key
FROM pg_class c
         JOIN pg_attribute a ON a.attrelid = c.oid
         LEFT JOIN pg_description b ON a.attrelid = b.objoid AND a.attnum = b.objsubid
         JOIN pg_type t ON a.atttypid = t.oid
         LEFT JOIN (
    SELECT
        kcu.column_name,
        con.constraint_type
    FROM information_schema.table_constraints con
             JOIN information_schema.key_column_usage kcu
                  ON con.constraint_name = kcu.constraint_name
    WHERE con.constraint_type = 'PRIMARY KEY'
      AND con.table_name = '%s'
) AS pk ON a.attname = pk.column_name
WHERE c.relname = '%s'
  AND a.attnum > 0
ORDER BY a.attnum;`, tableName, tableName)

	var fields PGFields
	result := db.Raw(query).Scan(&fields)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to get table fields: %v", result.Error)
	}

	return fields, nil
}

func getType(field *PGField) string {
	switch field.Type {
	case "character", "character varying", "varchar", "char", "bpchar":
		if field.Lengthvar > 4 {
			return fmt.Sprintf("varchar(%d)", field.Lengthvar-4)
		}
	}
	return field.Type
}

func closeDB(db *gorm.DB) {
	sqlDB, err := db.DB()
	if err != nil {
		return
	}
	_ = sqlDB.Close()
}
