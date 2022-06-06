package example

import (
	"github.com/stretchr/testify/assert"
	"log"
	"sync"
	"sync/atomic"
	"testing"
)

// 测试并发下单是否超时，扣减库存是否正常的
func TestConcurrentSubmitOrder(t *testing.T) {
	var (
		goroutineCount                = 100
		everyonePurchaseCount         = 3
		skuId                   int64 = 134930
		submitOrderSuccessCount int32 = 0
		wg                            = sync.WaitGroup{}
	)
	wg.Add(goroutineCount)
	for i := 0; i < goroutineCount; i++ {
		go func() {
			defer wg.Done()
			success, err := SubmitOrder(skuId, everyonePurchaseCount)
			if err != nil {
				log.Printf("submit order failed, err:%v \n", err)
				return
			}
			if success {
				atomic.AddInt32(&submitOrderSuccessCount, int32(everyonePurchaseCount))
			}
		}()
	}
	wg.Wait()
	assert.Equal(t, submitOrderSuccessCount, int32(getStock(skuId)))
	log.Println(getStock(skuId))
}
