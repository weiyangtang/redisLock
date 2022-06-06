package distributed_lock

import (
	"fmt"
	"github.com/go-redis/redis"
	"github.com/petermattis/goid"
	"github.com/stretchr/testify/assert"
	"log"
	"sync"
	"testing"
	"time"
)

var (
	redisClient = redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	})
)

func TestRedisLock_TryLock(t *testing.T) {
	var (
		lockName    = "inner_redis_lock_test"
		wg          = sync.WaitGroup{}
		lockExpired = 50 * time.Second
	)
	wg.Add(2)
	go func() {
		defer wg.Done()
		redisLocker, err := NewRedisLock(lockName, redisClient)
		assert.Nil(t, err)
		lock, err := redisLocker.TryLock(lockExpired)
		if err != nil {
			fmt.Println(err)
			return
		}
		if lock {
			fmt.Println("goroutine1 get lock")
			return
		}
		fmt.Println("goroutine1 not get lock")
	}()

	go func() {
		defer wg.Done()
		redisLocker, err := NewRedisLock(lockName, redisClient)
		assert.Nil(t, err)
		lock, err := redisLocker.TryLock(lockExpired)
		if err != nil {
			fmt.Println(err)
			return
		}
		if lock {
			fmt.Println("goroutine2 get lock")
			return
		}
		fmt.Println("goroutine2 not get lock")

	}()
	wg.Wait()
}

func TestReentrantLock(t *testing.T) {
	var (
		lockName    = "inner_redis_lock_test"
		lockExpired = 50 * time.Second
	)
	redisLocker, err := NewRedisLock(lockName, redisClient)
	assert.Nil(t, err)
	lock, err := redisLocker.TryLock(lockExpired)
	if err != nil {
		fmt.Println(err)
		return
	}
	if lock {
		fmt.Println("goroutine get lock")
	} else {
		fmt.Println("not get lock")
	}

	lock, err = redisLocker.TryLock(lockExpired)
	if err != nil {
		fmt.Println(err)
		return
	}
	if lock {
		fmt.Println("goroutine get lock")
		return
	} else {
		fmt.Println("not get lock")
	}

}

func TestLockAndUnlock(t *testing.T) {
	var (
		lockName    = "inner_redis_lock_test"
		wg          = sync.WaitGroup{}
		lockExpired = 50 * time.Second
	)
	wg.Add(2)
	go func() {
		defer wg.Done()
		fmt.Println(getGoroutineId())
		redisLocker, err := NewRedisLock(lockName, redisClient)
		assert.Nil(t, err)
		lock, err := redisLocker.TryLock(lockExpired)
		defer func() {
			unlock, err := redisLocker.Unlock()
			if err != nil {
				fmt.Println(err)
				return
			}
			fmt.Printf("goroutine1 unlock ret:%v \n", unlock)
		}()
		if err != nil {
			fmt.Println(err)
			return
		}
		if lock {
			fmt.Println("goroutine1 get lock")
			return
		}
		fmt.Println("goroutine1 not get lock")
	}()

	go func() {
		defer wg.Done()
		fmt.Println(getGoroutineId())
		time.Sleep(3 * time.Second)
		redisLocker, err := NewRedisLock(lockName, redisClient)
		assert.Nil(t, err)
		lock, err := redisLocker.TryLock(lockExpired)
		defer func() {
			unlock, err := redisLocker.Unlock()
			if err != nil {
				fmt.Println(err)
				return
			}
			fmt.Printf("goroutine2 unlock ret:%v \n", unlock)
		}()
		if err != nil {
			fmt.Println(err)
			return
		}
		if lock {
			fmt.Println("goroutine2 get lock")
			return
		}
		fmt.Println("goroutine2 not get lock")

	}()
	wg.Wait()
}

func TestGoroutineMisRelease(t *testing.T) {
	var (
		lockName    = "inner_redis_lock_test"
		wg          = sync.WaitGroup{}
		lockExpired = 500 * time.Second
	)
	wg.Add(2)
	go func() {
		defer wg.Done()
		fmt.Println(getGoroutineId())
		redisLocker, err := NewRedisLock(lockName, redisClient)
		assert.Nil(t, err)
		lock, err := redisLocker.TryLock(lockExpired)
		if err != nil {
			fmt.Println(err)
			return
		}
		if lock {
			fmt.Println("goroutine1 get lock")
			return
		}
		fmt.Println("goroutine1 not get lock")
	}()

	go func() {
		defer wg.Done()
		fmt.Println(getGoroutineId())
		time.Sleep(3 * time.Second)
		redisLocker, err := NewRedisLock(lockName, redisClient)
		assert.Nil(t, err)
		unlock, err := redisLocker.Unlock()
		if err != nil {
			fmt.Println(err)
			return
		}
		fmt.Println(unlock)
	}()
	wg.Wait()
}

// 阻塞式等待锁
func TestLockAndUnlockBlocked(t *testing.T) {
	var (
		lockName = "inner_redis_lock_test"
		wg       = sync.WaitGroup{}
	)
	wg.Add(2)
	go func() {
		defer wg.Done()
		log.Println("goroutine1 ", getGoroutineId())
		redisLocker, err := NewRedisLock(lockName, redisClient)
		assert.Nil(t, err)
		log.Println("goroutine1 start to get lock")
		err = redisLocker.Lock()
		log.Println("goroutine1 get lock")
		defer func() {
			unlock, err := redisLocker.Unlock()
			if err != nil {
				fmt.Println(err)
				return
			}
			log.Printf("goroutine1 unlock ret:%v \n", unlock)
		}()
		if err != nil {
			fmt.Println(err)
			return
		}
		log.Println("goroutine1 start 20s sleep")
		time.Sleep(20 * time.Second)
		log.Println("goroutine1 finish 20s sleep")
	}()

	go func() {
		defer wg.Done()
		log.Println("goroutine2 ", getGoroutineId())
		time.Sleep(3 * time.Second)
		redisLocker, err := NewRedisLock(lockName, redisClient)
		assert.Nil(t, err)
		log.Println("goroutine2 start to get lock")
		err = redisLocker.Lock()
		log.Println("goroutine2 get lock")
		defer func() {
			unlock, err := redisLocker.Unlock()
			if err != nil {
				fmt.Println(err)
				return
			}
			log.Printf("goroutine2 unlock ret:%v \n", unlock)
		}()
		if err != nil {
			fmt.Println(err)
			return
		}
		log.Println("goroutine2 start 20s sleep")
		time.Sleep(20 * time.Second)
		log.Println("goroutine2 finish 20s sleep")

	}()
	wg.Wait()
}

// 阻塞式等待锁
func TestLockAndUnlockWithTimeout(t *testing.T) {
	var (
		lockName = "inner_redis_lock_test"
		wg       = sync.WaitGroup{}
	)
	wg.Add(2)
	go func() {
		defer wg.Done()
		log.Println("goroutine1 ", getGoroutineId())
		redisLocker, err := NewRedisLock(lockName, redisClient)
		assert.Nil(t, err)
		log.Println("goroutine1 start to get lock")
		err = redisLocker.LockWithTimeout(8 * time.Second)
		if err != nil {
			log.Printf("goroutine1 get lock failed, err:%v \n", err)
			return
		}
		log.Println("goroutine1 get lock")
		defer func() {
			unlock, err := redisLocker.Unlock()
			if err != nil {
				fmt.Println(err)
				return
			}
			log.Printf("goroutine1 unlock ret:%v \n", unlock)
		}()
		log.Println("goroutine1 start 20s sleep")
		time.Sleep(20 * time.Second)
		log.Println("goroutine1 finish 20s sleep")
	}()

	go func() {
		defer wg.Done()
		log.Println("goroutine2 ", getGoroutineId())
		time.Sleep(3 * time.Second)
		redisLocker, err := NewRedisLock(lockName, redisClient)
		assert.Nil(t, err)
		log.Println("goroutine2 start to get lock")
		err = redisLocker.LockWithTimeout(8 * time.Second)
		if err != nil {
			log.Printf("goroutine2 get lock failed, err:%v \n", err)
			return
		}
		log.Println("goroutine2 get lock")
		defer func() {
			unlock, err := redisLocker.Unlock()
			if err != nil {
				fmt.Println(err)
				return
			}
			log.Printf("goroutine2 unlock ret:%v \n", unlock)
		}()
		log.Println("goroutine2 start 20s sleep")
		time.Sleep(20 * time.Second)
		log.Println("goroutine2 finish 20s sleep")

	}()
	wg.Wait()
}

func TestManyGoroutineGetLock(t *testing.T) {

	var (
		goroutineCount = 3
		wg             = sync.WaitGroup{}
		lockName       = "inner_redis_lock_test"
	)
	wg.Add(goroutineCount)
	for i := 0; i < goroutineCount; i++ {
		go func() {
			defer wg.Done()
			locker, err := NewRedisLock(lockName, redisClient)
			assert.Nil(t, err)
			locker.Lock()
			log.Printf("%v get lock", goid.Get())
			time.Sleep(1 * time.Second)
			locker.Unlock()
			log.Printf("%v  unlock", goid.Get())
		}()
	}
	wg.Wait()
}

func TestLuaCostTime(t *testing.T) {
	var (
		lockName = "inner_redis_lock_test"
	)
	startTime := time.Now()
	var lockLuaScript = redis.NewScript(lockLuaScript)
	lockLuaScript.Run(redisClient, []string{lockName, "goroutineId"}, 1, 4394902).Result()
	fmt.Println(time.Since(startTime))
}
