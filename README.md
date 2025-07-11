

## code snippets

1. time to live in golang redis
```go
// setting time to live in golang
package main

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

func main() {
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	ctx := context.Background()
	key := "mykey"
	value := "myvalue"
	expiration := 10 * time.Second // Set TTL to 10 seconds

	err := client.Set(ctx, key, value, expiration).Err()
	if err != nil {
		panic(err)
	}
	fmt.Printf("Key '%s' set with value '%s' and TTL of %s\n", key, value, expiration)
}
```

2. single flight in golang
```go
// use single flight to prevent cache stampede
func getDataSingleFlight(key string) (interface{}, error) {
	v, err, _ := g.Do(key, func() (interface{}, error) {
		// 查缓存
		data, err := getDataFromCache(key)
		if err == nil {
			return data, nil
		}
		if err == errNotFound {
			// 查DB
			data, err := getDataFromDB(key)
			if err == nil {
				setCache(data) // 设置缓存
				return data, nil
			}
			return nil, err
		}
		return nil, err // 缓存出错直接返回，防止灾难传递至DB
	})

	if err != nil {
		return nil, err
	}
	return v, nil
}
```

3. concrete implementation

key, value struct
```go
// value struct
// https://github.com/victorspringer/http-cache/blob/7d9f48f8ab9132cab212feee91e5097a9e603fa3/cache.go#L43
// Response is the cached response data structure.
type Response struct {
	// Value is the cached response value.
	Value []byte

	// Header is the cached response header.
	Header http.Header

	// Expiration is the cached response expiration date.
	Expiration time.Time

	// LastAccess is the last date a cached response was accessed.
	// Used by LRU and MRU algorithms.
	LastAccess time.Time

	// Frequency is the count of times a cached response is accessed.
	// Used for LFU and MFU algorithms.
	Frequency int
}

// key struct
// https://github.com/victorspringer/http-cache/blob/7d9f48f8ab9132cab212feee91e5097a9e603fa3/cache.go#L204C1-L217C2
import "hash/fnv"
func generateKey(URL string) uint64 {
	hash := fnv.New64a()
	hash.Write([]byte(URL))

	return hash.Sum64()
}

func generateKeyWithBody(URL string, body []byte) uint64 {
	hash := fnv.New64a()
	body = append([]byte(URL), body...)
	hash.Write(body)

	return hash.Sum64()
}


// this is the middleware implementation
// https://github.com/victorspringer/http-cache/blob/7d9f48f8ab9132cab212feee91e5097a9e603fa3/cache.go#L88
```

4. use with redis-cache

[repo link](https://github.com/go-redis/cache)

```go
package cache_test

import (
    "context"
    "fmt"
    "time"

    "github.com/redis/go-redis/v9"
    "github.com/go-redis/cache/v9"
)

type Object struct {
    Str string
    Num int
}

func Example_basicUsage() {
	// Ring is a Redis client that uses consistent hashing to distribute keys across multiple Redis servers (shards)
    ring := redis.NewRing(&redis.RingOptions{
        Addrs: map[string]string{
            "server1": ":6379",
            "server2": ":6380",
        },
    })

    mycache := cache.New(&cache.Options{
        Redis:      ring,
        LocalCache: cache.NewTinyLFU(1000, time.Minute),
    })

    ctx := context.TODO()
    key := "mykey"
    obj := &Object{
        Str: "mystring",
        Num: 42,
    }

    if err := mycache.Set(&cache.Item{
        Ctx:   ctx,
        Key:   key,
        Value: obj,
        TTL:   time.Hour,
    }); err != nil {
        panic(err)
    }

    var wanted Object
    if err := mycache.Get(ctx, key, &wanted); err == nil {
        fmt.Println(wanted)
    }

    // Output: {mystring 42}
}

func Example_advancedUsage() {
    ring := redis.NewRing(&redis.RingOptions{
        Addrs: map[string]string{
            "server1": ":6379",
            "server2": ":6380",
        },
    })

    mycache := cache.New(&cache.Options{
        Redis:      ring,
        LocalCache: cache.NewTinyLFU(1000, time.Minute),
    })

    obj := new(Object)
    err := mycache.Once(&cache.Item{
        Key:   "mykey",
        Value: obj, // destination
        Do: func(*cache.Item) (interface{}, error) {
            return &Object{
                Str: "mystring",
                Num: 42,
            }, nil
        },
    })
    if err != nil {
        panic(err)
    }
    fmt.Println(obj)
    // Output: &{mystring 42}
}
```

5. implement hierarchy cache as middleware on round tripper