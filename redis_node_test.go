package registry_goredis_test

import (
	"github.com/kordar/registry"
	registry_goredis "github.com/kordar/registry-goredis"
	"github.com/redis/go-redis/v9"
	"log"
	"testing"
	"time"
)

func TestRedisRegistry_Get(t *testing.T) {
	client := redis.NewUniversalClient(&redis.UniversalOptions{
		Addrs:    []string{"192.168.30.16:30202"},
		Password: "940430Dex",
		DB:       2,
	})

	var redisnoderegistry registry.Registry = registry_goredis.NewRedisNodeRegistry(client, &registry_goredis.RedisNodeRegistryOptions{
		Prefix:  "ABC",
		Node:    "123.12.34.2:3320",
		Timeout: time.Second * 30,
		Channel: "BOB",
		Reload: func(value []string, channel string) {
			log.Println("--------------", value, channel)
		},
		Heartbeat: time.Second * 3,
	})
	redisnoderegistry.Listener()
	time.Sleep(5 * time.Second)
	_ = redisnoderegistry.Register()

	registry2 := registry_goredis.NewRedisNodeRegistry(client, &registry_goredis.RedisNodeRegistryOptions{
		Prefix:  "ABC",
		Node:    "123.12.34.3:3320",
		Timeout: time.Second * 30,
		Channel: "BOB",
		Reload: func(value []string, channel string) {
			log.Println("22222222", value, channel)
		},
		Heartbeat: time.Second * 3,
	})
	registry2.Listener()
	_ = registry2.Register()

	time.Sleep(100 * time.Second)
}
