package dlock

import (
	"context"
	"errors"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/client/v3/concurrency"
)

var defaultTTL = 15 // seconds

type EtcdLock struct {
	session *concurrency.Session
	mutex   *concurrency.Mutex
}

// NewEtcd creates a new etcd locker with the given key and ttl.
func NewEtcd(client *clientv3.Client, key string, ttl int) (Locker, error) {
	if client == nil {
		return nil, errors.New("etcd client is nil")
	}

	if key == "" {
		return nil, errors.New("key is empty")
	}

	if ttl <= 0 {
		ttl = defaultTTL
	}
	expiration := time.Duration(ttl) * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), expiration)
	defer cancel()

	session, err := concurrency.NewSession(
		client,
		concurrency.WithTTL(ttl),
		concurrency.WithContext(ctx),
	)
	if err != nil {
		return nil, err
	}
	mutex := concurrency.NewMutex(session, key)

	locker := &EtcdLock{
		session: session,
		mutex:   mutex,
	}

	return locker, nil
}

// Lock blocks until the lock is acquired or the context is canceled.
func (l *EtcdLock) Lock(ctx context.Context) error {
	return l.mutex.Lock(ctx)
}

// Unlock releases the lock.
func (l *EtcdLock) Unlock(ctx context.Context) error {
	return l.mutex.Unlock(ctx)
}

// TryLock tries to acquire the lock without blocking.
func (l *EtcdLock) TryLock(ctx context.Context) (bool, error) {
	err := l.mutex.TryLock(ctx)
	if err == nil {
		return true, nil
	}
	if err == concurrency.ErrLocked {
		return false, nil
	}
	return false, err
}

// Close releases the lock and the etcd session.
func (l *EtcdLock) Close() error {
	if l.session != nil {
		return l.session.Close()
	}
	return nil
}
