package rediscache

// Ref.
// - SET command options : https://redis.io/commands/set/
// - PubSub & Key notify event : https://mattboodoo.com/2021/07/02/using-redis-keyspace-events-with-golang-for-a-poor-mans-event-driven-application/
// - Event config : https://redis.io/docs/manual/keyspace-notifications/

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

var (
	mainKeyPrefix               = "checkout"
	ctx                         = context.Background()
	rdb           *redis.Client = nil
)

func NewGoRedis() {
	rdb = redis.NewClient(&redis.Options{
		Addr:     "redis:6379",
		Password: "",
		DB:       0,
	})
}

func EnableKeyNotify() {
	// this is telling redis to publish events since it's off by default.
	_, err := rdb.Do(ctx, "CONFIG", "SET", "notify-keyspace-events", "KEA").Result()
	if err != nil {
		fmt.Printf("Unable to set keyspace events : %v\n", err.Error())
	} else {
		// this is telling redis to subscribe to events published in the keyevent channel,
		// specifically for expired events
		pubsub := rdb.PSubscribe(ctx, "__keyevent@0__:expired")
		wg := &sync.WaitGroup{}

		go func(redis.PubSub) {
			for {
				msg, err := pubsub.ReceiveMessage(ctx)
				if err != nil {
					fmt.Printf("[PubSub] Error message : %v\n", err.Error())
					break
				}
				fmt.Printf("[PubSub] Keyspace event recieved : %v\n", msg.String())

				// >>>>>> DO SOMETHING HERE AFTER NOTIFIED <<<<<<<
			}
		}(*pubsub)

		wg.Wait()
	}
}

func setNxTTL(key string) {
	result, err := rdb.SetNX(ctx, key, "", redis.KeepTTL).Result()
	if err == redis.Nil {
		fmt.Printf("'%s' does not exist!\n", key)
	} else if err != nil {
		fmt.Printf("Set TTL error : %s\n", err.Error())
	} else {
		fmt.Printf("Set TTL : %v\n", result)
	}
}

func convTTLStatusToMsg(ttl int64) string {
	msg := ""
	if ttl == -2 {
		msg = "Key does not exist."
	} else if ttl == -1 {
		msg = "Key exists but has no associated expire."
	} else {
		msg = fmt.Sprintf("Remaining %v (in seconds)", ttl)
	}
	return msg
}

func GetOrder(keyPrefix string, id string) ResponseObj {
	res := ResponseObj{
		Status: http.StatusBadGateway,
		Error:  nil,
		Data:   nil,
	}

	key := fmt.Sprintf("%s:%s", keyPrefix, id)

	pipe := rdb.Pipeline()
	info := pipe.Get(ctx, key)
	ttl := pipe.Do(ctx, "TTL", key)

	_, errPipe := pipe.Exec(ctx)
	if errPipe != nil {
		result, ttlErr := rdb.Do(ctx, "TTL", key).Result()
		if ttlErr == redis.Nil {
			errMsg := fmt.Sprintf("[GetOrder] Got redis nil in : %s", ttlErr.Error())
			res.Error = &errMsg
			res.Status = http.StatusBadRequest
		} else if ttlErr != nil {
			errMsg := fmt.Sprintf("[GetOrder] Get TTL error : %s", ttlErr.Error())
			res.Error = &errMsg
			res.Status = http.StatusBadRequest
		} else {
			val := fmt.Sprintf("%v", result)
			ttl, _ := strconv.ParseInt(val, 10, 64)
			res.Data = &Info{
				TTL:   ttl,
				Value: convTTLStatusToMsg(ttl),
			}
			res.Status = http.StatusOK
		}
	} else {
		ttlVal, _ := ttl.Int64()
		res.Data = &Info{
			TTL:   ttlVal,
			Value: info.Val(),
		}
		res.Status = http.StatusOK
	}

	return res
}

func Checkout(id string, expire int64) ResponseObj {
	res := ResponseObj{
		Status: http.StatusBadGateway,
		Error:  nil,
		Data:   nil,
	}

	if expire < 1 {
		expire = 1
	}

	expireInMinutes := time.Duration(expire) * time.Second
	key := fmt.Sprintf("%s:%s", mainKeyPrefix, id)

	setNxTTL(mainKeyPrefix)

	// Set key value with expire in seconds
	// if the key does not exist
	result, err := rdb.SetNX(ctx, key, id, expireInMinutes).Result()
	if err == redis.Nil {
		errMsg := fmt.Sprintf("'%s' does not exist!", id)
		res.Error = &errMsg
		res.Status = http.StatusBadRequest
	} else if err != nil {
		errMsg := fmt.Sprintf("Checkout error : %s", err.Error())
		res.Error = &errMsg
		res.Status = http.StatusBadRequest
	} else {
		msg := fmt.Sprintf("Set checkout ID : %v", result)
		res.Data = &Info{Value: msg}
		res.Status = http.StatusOK
	}

	return res
}
