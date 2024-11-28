# registry-goredis

基于`redis`的注册中心实现，工作流如下：


1、节点启动后向`redis`注册节点基本信息

```go
ctx := context.Background()
data := map[string]string{
    "node":         r.options.Node,
    "refresh_time": time.Now().Format("2006-01-02 15:04:05"),
    "status":       "online",
    "weight":       r.options.Weight,
}
marshal, _ := json.Marshal(&data)
return r.client.Set(ctx, r.key(), string(marshal), r.options.Timeout).Err()
```

2、当前节点注册成功后，广播`reload`事件，通知其他节点进行刷新最新节点信息（注意相同类型节点需要订阅同一个频道）

```go
PubMessage(r.client, r.options.Channel, "reload")
```

3、节点启动后同时对频道进行监听

```go
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
```

4、`reload`操作获取最新节点列表信息（注：节点删除操作会触发本地`reload`）。

***基于上述工作流，本地可维护所有节点列表信息，本地程序通过该列表可实现相应功能。例如，使用一致性hash算法计算列表，实现分布式定时任务调度等。***

**节点异常退出如何通知其他节点?**

> 节点启动时插入心跳定时任务触发`reload`操作，异常退出的节点在`redis`超时后会自动被剔除，本地获取的最新列表将会自动剔除该节点。

```go
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
```

**是否支持频繁节点增删操作？**

首先该系统维护的节点列表遵循最终一致性原则，在节点增删过程中可能会产生短暂的节点列表不一致问题，但随着时间的推移，系统各个节点将达到最终列表一致。

如果频繁产生节点增删操作，系统可能会长期处于同步操作中，这种情况下可以考虑使用`zookeeper`实现强一致性系统。