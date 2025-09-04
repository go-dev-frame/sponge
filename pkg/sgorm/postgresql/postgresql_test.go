package postgresql

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestInit(t *testing.T) {
	dsn := "host=192.168.3.37 user=root password=123456 dbname=account port=5432 sslmode=disable TimeZone=Asia/Shanghai"
	db, err := Init(dsn, WithEnableTrace())
	if err != nil {
		// ignore test error about not being able to connect to real postgresql
		t.Logf("connect to postgresql failed, err=%v, dsn=%s", err, dsn)
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
