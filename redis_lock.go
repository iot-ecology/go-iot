package main

import (
	"context"
	"github.com/redis/go-redis/v9"
	"sync"
	"time"

	"github.com/google/uuid"
)

const (
	LockTime         = 4 * time.Second
	RS_DISTLOCK_NS   = "tdln:"
	RELEASE_LOCK_LUA = `
        if redis.call('get',KEYS[1])==ARGV[1] then
            return redis.call('del', KEYS[1])
        else
            return 0
        end
    `
)

type RedisDistLock struct {
	id          string
	lockName    string
	redisClient *redis.Client
	m           sync.Mutex
}

func NewRedisDistLock(redisClient *redis.Client, lockName string) *RedisDistLock {
	return &RedisDistLock{
		lockName:    lockName,
		redisClient: redisClient,
	}
}

func (this *RedisDistLock) Lock() {
	for !this.TryLock() {
		time.Sleep(5 * time.Second)
	}
}

func (this *RedisDistLock) TryLock() bool {
	if this.id != "" {
		// 处于加锁中
		return false
	}
	this.m.Lock()
	defer this.m.Unlock()
	if this.id != "" {
		// 处于加锁中
		return false
	}
	ctx := context.Background()
	id := uuid.New().String()
	reply := this.redisClient.SetNX(ctx, RS_DISTLOCK_NS+this.lockName, id, LockTime)
	if reply.Err() == nil && reply.Val() {
		this.id = id
		return true
	}

	return false
}

func (this *RedisDistLock) Unlock() {
	if this.id == "" {
		// 未加锁
		panic("解锁失败，因为未加锁")
	}
	this.m.Lock()
	defer this.m.Unlock()
	if this.id == "" {
		// 未加锁
		panic("解锁失败，因为未加锁")
	}
	ctx := context.Background()
	reply := this.redisClient.Eval(ctx, RELEASE_LOCK_LUA, []string{RS_DISTLOCK_NS + this.lockName}, this.id)
	if reply.Err() != nil {
		panic("释放锁失败！")
	} else {
		this.id = ""
	}
}
