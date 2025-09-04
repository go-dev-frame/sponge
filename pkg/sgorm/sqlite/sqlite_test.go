package sqlite

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestInit(t *testing.T) {
	dbFile := "test_sqlite.db"
	db, err := Init(dbFile)
	if err != nil {
		// ignore test error about not being able to connect to real sqlite
		t.Logf("connect to sqlite failed, err=%v, dbFile=%s", err, dbFile)
		return
	}
	defer Close(db)

	t.Logf("%+v", db.Name())
}

func Test_gormConfig(t *testing.T) {
	o := defaultOptions()
	o.apply(
		WithLogging(nil),
		WithLogging(nil, 4),
		WithSlowThreshold(time.Millisecond*100),
		WithEnableTrace(),
		WithMaxIdleConns(5),
		WithMaxOpenConns(50),
		WithConnMaxLifetime(time.Minute*3),
		WithEnableForeignKey(),
		WithLogRequestIDKey("request_id"),
		WithGormPlugin(nil),
	)

	c := gormConfig(o)
	assert.NotNil(t, c)
}
