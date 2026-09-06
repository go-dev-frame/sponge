// Package sgorm is a library encapsulated on gorm.io/gorm
package sgorm

import (
	"reflect"
	"time"

	"github.com/huandu/xstrings"
	"gorm.io/gorm"

	"github.com/go-dev-frame/sponge/pkg/sgorm/dbclose"
)

type DB = gorm.DB

var ErrRecordNotFound = gorm.ErrRecordNotFound

const (
	// DBDriverMysql mysql driver
	DBDriverMysql = "mysql"
	// DBDriverPostgresql postgresql driver
	DBDriverPostgresql = "postgresql"
	// DBDriverTidb tidb driver
	DBDriverTidb = "tidb"
	// DBDriverSqlite sqlite driver
	DBDriverSqlite = "sqlite"
)

// Model embedded structs, add `gorm: "embedded"` when defining table structs
type Model struct {
	ID        uint64         `gorm:"column:id;AUTO_INCREMENT;primary_key" json:"id"`
	CreatedAt time.Time      `gorm:"column:created_at" json:"createdAt"`
	UpdatedAt time.Time      `gorm:"column:updated_at" json:"updatedAt"`
	DeletedAt gorm.DeletedAt `gorm:"column:deleted_at;index" json:"-"`
}

// Model2 embedded structs, json tag named is snake case
type Model2 struct {
	ID        uint64         `gorm:"column:id;AUTO_INCREMENT;primary_key" json:"id"`
	CreatedAt time.Time      `gorm:"column:created_at" json:"created_at"`
	UpdatedAt time.Time      `gorm:"column:updated_at" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"column:deleted_at;index" json:"-"`
}

// KV map type
type KV = map[string]interface{}

// GetTableName get table name
func GetTableName(object interface{}) string {
	tableName := ""

	typeof := reflect.TypeOf(object)
	switch typeof.Kind() {
	case reflect.Pointer:
		tableName = typeof.Elem().Name()
	case reflect.Struct:
		tableName = typeof.Name()
	default:
		return tableName
	}

	return xstrings.ToSnakeCase(tableName)
}

// CloseDB close db
func CloseDB(db *gorm.DB) error {
	return dbclose.Close(db)
}
