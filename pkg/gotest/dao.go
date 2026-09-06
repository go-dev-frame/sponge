package gotest

import (
	"context"
	"database/sql/driver"
	"reflect"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

// Dao dao info
type Dao struct {
	Ctx      context.Context
	TestData interface{}
	SQLMock  sqlmock.Sqlmock
	Cache    *Cache
	DB       *gorm.DB
	IDao     interface{}
	AnyTime  *anyTime
	closeFns []func()
}

// NewDao instantiated dao
func NewDao(c *Cache, testData interface{}) *Dao {
	var closeFns []func()

	if c != nil {
		closeFns = append(closeFns, func() {
			c.redisServer.Close()
		})
	}

	// mock mysql
	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		panic(err)
	}
	mockDB, err := gorm.Open(
		mysql.New(mysql.Config{
			Conn:                      sqlDB,
			SkipInitializeWithVersion: true,
		}),
		&gorm.Config{
			NamingStrategy: schema.NamingStrategy{SingularTable: true},
		})
	if err != nil {
		panic(err)
	}

	closeFns = append(closeFns, func() {
		_ = sqlDB.Close()
	})

	return &Dao{
		Ctx:      context.Background(),
		SQLMock:  mock,
		TestData: testData,
		Cache:    c,
		DB:       mockDB,
		AnyTime:  &anyTime{},
		closeFns: closeFns,
	}
}

// Close dao
func (d *Dao) Close() {
	for _, fn := range d.closeFns {
		fn()
	}
}

// GetAnyArgs Dynamic generation of parameter types based on structures
func (d *Dao) GetAnyArgs(obj interface{}) []driver.Value {
	to := reflect.TypeOf(obj)
	vo := reflect.ValueOf(obj)

	//nolint
	if to.Kind() == reflect.Pointer {
		if vo.IsNil() {
			panic("nil ptr")
		}

		originType := reflect.ValueOf(d.TestData).Elem().Type()
		if originType.Kind() != reflect.Struct {
			return nil
		}

		to = to.Elem()
		vo = vo.Elem()
	} else {
		panic("non ptr")
	}

	num := to.NumField()
	count := 0
	for i := 0; i < num; i++ {
		field := to.Field(i)
		fieldName := field.Name
		fieldValue := vo.FieldByName(fieldName)
		if !fieldValue.IsValid() {
			continue
		}
		if fieldValue.CanInterface() {
			if fieldValue.Kind() == reflect.Struct && field.Type.String() != "time.Time" {
				count += fieldValue.NumField()
				continue
			}
			count++
		}
	}

	var args []driver.Value
	for i := 0; i < count; i++ {
		args = append(args, sqlmock.AnyArg())
	}

	return args
}

type anyTime struct{}

// Match satisfies sqlmock.Argument interface,
// if the table has fields of type time.Time, this method must be implemented
func (a *anyTime) Match(v driver.Value) bool {
	_, ok := v.(time.Time)
	return ok
}
