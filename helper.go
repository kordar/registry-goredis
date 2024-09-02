package registry_goredis

import (
	"context"
	"errors"
	"github.com/redis/go-redis/v9"
)

type Event struct {
	Channel string
	Fn      func(payload string, channel string)
}

func PubMessage(c redis.UniversalClient, channel, msg string) error {
	ctx := context.Background()
	cmd := c.Publish(ctx, channel, msg)
	return cmd.Err()
}

func SubMessage(c redis.UniversalClient, events ...Event) error {
	ctx := context.Background()
	if events == nil || len(events) == 0 {
		return errors.New("subscribe channel fail")
	}

	fn := map[string]func(payload string, channel string){}
	channels := make([]string, 0)
	for _, event := range events {
		channels = append(channels, event.Channel)
		fn[event.Channel] = event.Fn
	}

	pubsub := c.Subscribe(ctx, channels...)
	_, err := pubsub.Receive(ctx)
	if err != nil {
		return err
	}

	ch := pubsub.Channel()
	for msg := range ch {
		fn[msg.Channel](msg.Payload, msg.Channel)
	}

	return nil
}
