package dlock

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/go-dev-frame/sponge/pkg/goredis"
)

func TestRedisLock_TryLock(t *testing.T) {
	initLocker := func() Locker {
		return getRedisLock()
	}
	testLockAndUnlock(initLocker, false, t)
}

func TestRedisLock_Lock(t *testing.T) {
	initLocker := func() Locker {
		return getRedisLock()
	}
	testLockAndUnlock(initLocker, true, t)
}

func TestClusterRedis_TryLock(t *testing.T) {
	initLocker := func() Locker {
		return getClusterRedisLock()
	}
	testLockAndUnlock(initLocker, false, t)
}

func TestClusterRedis_Lock(t *testing.T) {
	initLocker := func() Locker {
		return getClusterRedisLock()
	}
	testLockAndUnlock(initLocker, true, t)
}

func getRedisLock() Locker {
	redisCli, err := goredis.Init("default:123456@127.0.0.1:6379")
	if err != nil {
		fmt.Println(err)
		return nil
	}

	locker, err := NewRedisLock(redisCli, "test_lock")
	if err != nil {
		return nil
	}
	return locker
}

func getClusterRedisLock() Locker {
	addrs := []string{"127.0.0.1:6380", "127.0.0.1:6381", "127.0.0.1:6382"}
	clusterClient, err := goredis.InitCluster(addrs, "", "123456")
	if err != nil {
		fmt.Println(err)
		return nil
	}

	locker, err := NewRedisClusterLock(clusterClient, "test_cluster_lock")
	if err != nil {
		return nil
	}
	return locker
}

func testLockAndUnlock(initLocker func() Locker, isBlock bool, t *testing.T) {
	waitGroup := &sync.WaitGroup{}
	for i := 1; i <= 10; i++ {
		waitGroup.Add(1)
		go func(i int) {
			defer waitGroup.Done()
			ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
			defer cancel()
			NO := fmt.Sprintf("[NO-%d] ", i)

			locker := initLocker()
			if locker == nil {
				t.Log("logger init failed")
				return
			}

			var err error
			var ok bool
			for {
				select {
				case <-ctx.Done():
					return
				default:
				}

				time.Sleep(time.Millisecond * 50)
				if isBlock {
					err = locker.Lock(ctx)
					if err == nil {
						ok = true
					}
				} else {
					ok, err = locker.TryLock(ctx)
				}
				if err != nil {
					//t.Log(NO+"try lock error:", err)
					continue
				}
				if ok {
					t.Log(NO + "acquire lock success, and do something")
					time.Sleep(time.Millisecond * 200)
					err = locker.Unlock(ctx)
					if err != nil {
						return
					}
					t.Log(NO + "unlock done")
					return
				}
			}
		}(i)
	}

	waitGroup.Wait()
}
