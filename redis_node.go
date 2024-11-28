package registry_goredis

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/redis/go-redis/v9"
	"sync"
	"time"
)

type RedisNodeRegistryOptions struct {
	Prefix    string
	Node      string
	Timeout   time.Duration
	Channel   string
	Heartbeat time.Duration
	Reload    func(data []string, channel string)
	Weight    string
}

// RedisNodeRegistry redis节点注册
type RedisNodeRegistry struct {
	options   *RedisNodeRegistryOptions
	once      sync.Once
	heartOnce sync.Once
	client    redis.UniversalClient
}

func NewRedisNodeRegistry(client redis.UniversalClient, options *RedisNodeRegistryOptions) *RedisNodeRegistry {
	return &RedisNodeRegistry{
		client:    client,
		once:      sync.Once{},
		heartOnce: sync.Once{},
		options:   options,
	}
}

func (r *RedisNodeRegistry) key() string {
	return fmt.Sprintf("%s:%s", r.options.Prefix, r.options.Node)
}

func (r *RedisNodeRegistry) Get() (interface{}, error) {
	ctx := context.Background()
	keys := r.client.Keys(ctx, r.options.Prefix+":*")
	cmd := r.client.MGet(ctx, keys.Val()...)
	if cmd.Err() != nil {
		return nil, cmd.Err()
	}
	return cmd.Val(), nil
}

// Remove 移除配置
func (r *RedisNodeRegistry) Remove() error {
	ctx := context.Background()
	if err := r.client.Del(ctx, r.key()).Err(); err == nil {
		return PubMessage(r.client, r.options.Channel, "reload")
	} else {
		return err
	}
}

func (r *RedisNodeRegistry) regRemote() error {
	ctx := context.Background()
	data := map[string]string{
		"node":         r.options.Node,
		"refresh_time": time.Now().Format("2006-01-02 15:04:05"),
		"status":       "online",
		"weight":       r.options.Weight,
	}
	marshal, _ := json.Marshal(&data)
	return r.client.Set(ctx, r.key(), string(marshal), r.options.Timeout).Err()
}

// Register 将节点信息写入到redis中,并向订阅者进行通知
func (r *RedisNodeRegistry) Register() error {
	r.heartOnce.Do(func() {
		ticker := time.NewTicker(r.options.Heartbeat)
		go func() {
			for {
				select {
				case <-ticker.C:
					r.reload(r.options.Channel)
					_ = r.regRemote()
					break
				}
			}
		}()
	})

	if err := r.regRemote(); err == nil {
		return PubMessage(r.client, r.options.Channel, "reload")
	} else {
		return nil
	}
}

func (r *RedisNodeRegistry) Listener() {
	r.once.Do(func() {
		go SubMessage(r.client, Event{
			Channel: r.options.Channel,
			Fn: func(payload string, channel string) {
				if payload == "reload" {
					r.reload(channel)
				}
			},
		})
	})

}

func (r *RedisNodeRegistry) reload(channel string) {
	data := make([]string, 0)
	if val, err := r.Get(); err == nil {
		vv := val.([]interface{})
		for _, v := range vv {
			data = append(data, v.(string))
		}
		if r.options.Reload != nil {
			r.options.Reload(data, channel)
		}
	}
}
