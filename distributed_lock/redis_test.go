package distributed_lock

import (
	"fmt"
	"github.com/go-redis/redis"
	"github.com/stretchr/testify/assert"
	"sync"
	"testing"
	"time"
)

func TestLuaExec(t *testing.T) {
	const (
		luaShell = `
if redis.call("exists", KEYS[1]) == 0 then
    redis.call("hset", KEYS[1], KEYS[2], ARGV[1])
    redis.call("pexpire", KEYS[1], ARGV[2])
    return 0;
else
    if redis.call("hexists", KEYS[1],KEYS[2]) == 0 then
        return redis.call("pttl", KEYS[1]);
    else
        redis.call("hincrby", KEYS[1], KEYS[2], 1)
        redis.call("pexpire", KEYS[1], ARGV[2])
        return 0;
    end
end`
	)

	rdb := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	})
	//result, err := rdb.Eval(luaShell, []string{"redis-lock", "34u38493"}, 1, 5000).Result()
	//assert.Nil(t, err)
	//fmt.Println(result)
	var updateRecordExpireScript = redis.NewScript(luaShell)
	result, err := updateRecordExpireScript.Run(rdb, []string{"redis-lock", "34893849" +
		"" +
		"" +
		""}, 1, 500000).Result()
	assert.Nil(t, err)
	fmt.Println(result)

}

func TestSimpleLua(t *testing.T) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	})
	const (
		simpleLua = `return redis.call("exists", KEYS[1])`
	)

	result, err := rdb.Eval(simpleLua, []string{"redis-lock", "34u38493"}, 1, 5000).Result()
	assert.Nil(t, err)
	fmt.Println(result)
	rdb.Subscribe("").Channel()
}

func TestSubscribe(t *testing.T) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	})
	channelName := "subscribe_channel_test"
	wg := sync.WaitGroup{}
	wg.Add(2)
	go func() {
		for {
			var msg = <-rdb.Subscribe(channelName).Channel()
			fmt.Println(msg)
			//time.Sleep(1 * time.Second)
		}
	}()

	go func() {
		const (
			simpleLua = `return redis.call("publish", KEYS[1], ARGV[1])`
		)
		var i = 1
		for {
			rdb.Eval(simpleLua, []string{channelName}, i).Result()
			//fmt.Println(result)
			i++
			time.Sleep(1 * time.Second)
		}
	}()
	wg.Wait()
}
