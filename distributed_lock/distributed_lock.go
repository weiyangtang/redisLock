package distributed_lock

import (
	"errors"
	"fmt"
	"github.com/go-redis/redis"
	"github.com/gogf/gf/v2/util/gconv"
	"github.com/petermattis/goid"
	"log"
	"net"
	"strings"
	"sync"
	"time"
)

const (
	defaultExpiredTime           = 15 * time.Second // 锁默认过期时间 15s
	defaultAutoRenewExpiredDelta = 5 * time.Second  // 自动续期执行的间隔时间
)

const (
	lockKeyTemplate string = "redis_lock_%s"

	unlockChanName string = "%s_unlock_chan"
)

// 实现了golang 语言版本的redis分布式锁
//  1. 自动续期
//  2. 可重入
//  3. 支持阻塞式等待获取锁
//  4. 支持最大超时等待时间

// RedisLock 开发思路：先实现支持可重入的非阻塞式不会自动续期分布式锁
// 锁的数据结构 hash: key: lock_name, subKey: host_goId, subVal: reentrantCount 重入次数
// lock() 加锁
// exists lock_name 锁是否存在
//  - 0 不存在，设置锁的拥有锁的goroutine和重入次数为1 hset lock_name host_goId 1
//  - 1 存在，
//		- 判断拥有锁的goroutine是否为当前协程，直接判断key: lock_name subKey: host_goId 是否存在 hexists lock_name host_goId
//			- 0 不存在，其他协程已拥有锁
//          - 1 存在
//            	- 重入次数加1，hincrby lock_name host_goId 1
//
// unlock() 释放锁
// hexists lock_name host_goId 判断当前协程是否拥有锁
// - 0 不存在，不一定要报错
// - 1 存在
// 		- 释放锁 hincrby lock_name host_goId -1
// 		- 重入次数-1后 > 0, 说明还有当前协程还有重入
//		- 重入次数-1后 <= 0, 释放锁 del lock_name
type RedisLock interface {
	// LockWithTimeout 阻塞式获取分布式锁，设置获取锁超时时间
	LockWithTimeout(timeout time.Duration) error

	// Lock 阻塞式获取锁，不设置超时时间
	Lock() error

	// TryLock 尝试获取锁
	// @return
	// bool getLock true:获取到锁
	// error : 获取锁的error
	TryLock(expiredTime time.Duration) (bool, error)

	// Unlock 释放锁
	Unlock() (bool, error)
}

var redisLockInst RedisLock

func NewRedisLock(lockName string, redisClient *redis.Client) (RedisLock, error) {
	if strings.TrimSpace(lockName) == "" {
		return nil, errors.New("lockName cannot empty")
	}
	if redisClient == nil {
		return nil, errors.New("redisClient cannot nil")
	}
	instance := &redisLock{
		lockName:           fmt.Sprintf(lockKeyTemplate, lockName),
		unlockMsgChanName:  fmt.Sprintf(unlockChanName, lockName),
		redisClient:        redisClient,
		mutex:              sync.Mutex{},
		autoReExpiredDelta: defaultAutoRenewExpiredDelta,
		defaultExpiredTime: defaultExpiredTime,
	}
	go instance.autoRenewExpire()
	redisLockInst = instance
	return redisLockInst, nil
}

type redisLock struct {
	lockName           string
	unlockMsgChanName  string
	mutex              sync.Mutex
	redisClient        *redis.Client
	autoReExpiredDelta time.Duration // redis锁自动设置过期时间的执行间隔
	defaultExpiredTime time.Duration // 默认过期时间
	goroutineId        string
}

func (r *redisLock) LockWithTimeout(timeout time.Duration) error {
	waitChan := make(chan struct{}, 1)
	exceedTime := false
	ticker := time.NewTicker(timeout)
	go func() {
		for {
			select {
			case <-r.redisClient.Subscribe(r.unlockMsgChanName).Channel():
				waitChan <- struct{}{}
			case <-ticker.C:
				exceedTime = true
				waitChan <- struct{}{}
			}
		}
	}()
	for {
		if exceedTime {
			return errors.New("exceed lock wait timeout")
		}
		lock, err := r.TryLock(r.defaultExpiredTime)
		if err != nil {
			return err
		}
		if lock {
			return nil
		}
		<-waitChan
	}
}

func (r *redisLock) Lock() error {
	waitChan := make(chan struct{}, 1)
	go func() {
		for {
			select {
			case <-r.redisClient.Subscribe(r.unlockMsgChanName).Channel():
				waitChan <- struct{}{}
			}
		}
	}()
	for {
		lock, err := r.TryLock(r.defaultExpiredTime)
		if err != nil {
			return err
		}
		if lock {
			return nil
		}
		<-waitChan
	}
}

func (r *redisLock) TryLock(expiredTime time.Duration) (bool, error) {
	expiredMilliseconds := int64(expiredTime / time.Millisecond)
	result, err := r.execLockLua(r.lockName, getGoroutineId(), expiredMilliseconds)
	if err != nil {
		return false, err
	}

	if result == 0 {
		return true, nil
	}

	return false, nil
}

func (r *redisLock) Unlock() (bool, error) {
	return r.execUnlockLua(r.lockName, getGoroutineId(), r.unlockMsgChanName)
}

func (r *redisLock) execLockLua(lockName string, goroutineId string, timeoutMilliseconds int64) (int, error) {
	var lockLuaScript = redis.NewScript(lockLuaScript)
	result, err := lockLuaScript.Run(r.redisClient, []string{lockName, goroutineId}, 1, timeoutMilliseconds).Result()
	return gconv.Int(result), err
}

func (r *redisLock) execUnlockLua(lockName string, goroutineId string, unlockChanName string) (bool, error) {
	var lockLuaScript = redis.NewScript(unlockLuaScript)
	res, err := lockLuaScript.Run(r.redisClient, []string{lockName, goroutineId}, unlockChanName).Result()
	if err != nil {
		return false, err
	}
	if gconv.Int(res) > 0 {
		return true, nil
	}
	return false, nil
}

func (r *redisLock) autoRenewExpire() {
	ticker := time.NewTicker(r.autoReExpiredDelta)
	for {
		select {
		case <-ticker.C:
			if err := r.redisClient.PExpire(r.lockName, r.defaultExpiredTime).Err(); err != nil {
				log.Printf("renew expired failed, lock name:%s,err:%v \n", r.lockName, err)
				continue
			}
		}
	}
}

func getGoroutineId() string {
	// host-goroutineId
	goroutineId := goid.Get()
	ip, err := getLocalIp()
	if err != nil {
		return ""
	}

	return fmt.Sprintf("%v_%v", ip, goroutineId)
}

func getLocalIp() (string, error) {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "", err
	}
	for _, address := range addrs {
		// 检查ip地址判断是否回环地址
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String(), nil
			}
		}
	}
	return "", nil
}
