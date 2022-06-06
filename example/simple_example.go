package example

import (
	"errors"
	"fmt"
	"github.com/go-redis/redis"
	"github.com/weiyang_tang/redisLock/distributed_lock"
	"time"
)

// 下单场景下的分布式锁使用

var (
	stock             = 30
	skuId       int64 = 134930
	redisClient       = redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	})
)

func getStock(skuId int64) int {
	return stock
}

func decreaseStock(skuId int64, purchasesCount int) int {
	stock = stock - purchasesCount
	return stock
}

// 模拟下单接口
func SubmitOrder(skuId int64, purchasesCount int) (bool, error) {
	lockName := fmt.Sprintf("skuId:%d:submitOrder:lock", skuId)
	redisLock, err := distributed_lock.NewRedisLock(lockName, redisClient)
	if err != nil {
		return false, err
	}
	redisLock.Lock()
	if getStock(skuId) < purchasesCount {
		return false, nil
	}
	decreaseStock(skuId, purchasesCount)
	// other cost times operation
	time.Sleep(10 * time.Millisecond)
	unlock, err := redisLock.Unlock()
	if err != nil {
		return false, err
	}
	if !unlock {
		return false, errors.New(fmt.Sprintf("unlock error %v", err))
	}
	return true, nil
}
