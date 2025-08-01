package database

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/go-dev-frame/sponge/configs"
	"github.com/go-dev-frame/sponge/pkg/sgorm"
	"github.com/go-dev-frame/sponge/pkg/utils"

	"github.com/go-dev-frame/sponge/internal/config"
)

func TestGetDB(t *testing.T) {
	err := config.Init(configs.Location("serverNameExample.yml"))
	if err != nil {
		panic(err)
	}

	config.Get().App.EnableTrace = true
	config.Get().Database.Mysql.EnableLog = true

	time.Sleep(time.Millisecond * 10)
	err = CloseDB()
	assert.NoError(t, err)

	utils.SafeRunWithTimeout(time.Second*2, func(cancel context.CancelFunc) {
		db := GetDB()
		assert.NotNil(t, db)
		cancel()
	})
}

func TestInitMysqlError(t *testing.T) {
	err := config.Init(configs.Location("serverNameExample.yml"))
	if err != nil {
		panic(err)
	}

	// change config error test
	config.Get().Database.Mysql.Dsn = "root:123456@(127.0.0.1:3306)/test"

	utils.SafeRunWithTimeout(time.Second*2, func(cancel context.CancelFunc) {
		_ = CloseDB()
		InitMysql()
		assert.NotNil(t, gdb)
		cancel()
	})
}

func TestInitPostgresqlError(t *testing.T) {
	_ = config.Init(configs.Location("serverNameExample.yml"))

	// change config error test
	config.Get().Database.Postgresql.Dsn = "root:123456@(127.0.0.1:5432)/test"

	utils.SafeRunWithTimeout(time.Second*2, func(cancel context.CancelFunc) {
		_ = CloseDB()
		InitPostgresql()
		assert.NotNil(t, gdb)
		cancel()
	})
}

func TestInitSqliteError(t *testing.T) {
	_ = config.Init(configs.Location("serverNameExample.yml"))
	utils.SafeRunWithTimeout(time.Second*2, func(cancel context.CancelFunc) {
		InitSqlite()
		assert.NotNil(t, gdb)
		cancel()
	})
}

func TestCloseDB(t *testing.T) {
	defer func() { recover() }()
	gdb = &sgorm.DB{}
	err := CloseDB()
	assert.NoError(t, err)
}

func TestInitRedis(t *testing.T) {
	defer func() {
		if e := recover(); e != nil {
			t.Log("ignore connect redis error info")
		}
	}()

	err := config.Init(configs.Location("serverNameExample.yml"))
	if err != nil {
		panic(err)
	}

	InitRedis()
	cli := GetRedisCli()
	assert.NotNil(t, cli)
	time.Sleep(time.Millisecond * 10)
	err = CloseRedis()
	assert.NoError(t, err)

	// change config error test
	config.Get().App.EnableTrace = true
	config.Get().Redis.Dsn = "default:123456@127.0.0.1:6379/0"
	redisCli = nil
	_ = CloseRedis()
	_ = GetRedisCli()
}

func TestGetCacheType(t *testing.T) {
	InitCache("memory")
	ct := GetCacheType()
	assert.NotNil(t, ct)

	err := config.Init(configs.Location("serverNameExample.yml"))
	if err != nil {
		panic(err)
	}

	cacheType = nil
	defer func() { recover() }()
	ct = GetCacheType()
	assert.NotNil(t, ct)

	defer func() { recover() }()
	InitCache("redis")
	ct = GetCacheType()
	assert.NotNil(t, ct)
}
