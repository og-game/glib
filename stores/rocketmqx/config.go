package rocketmqx

import (
	"context"
	"github.com/apache/rocketmq-clients/golang"
)

type Config struct {
	ConsoleAppenderEnabled bool           `json:"console_appender_enabled,default=true"`
	Endpoint               string         `json:"endpoint"`
	AccessKey              string         `json:"access_key,optional"`
	AccessSecret           string         `json:"access_secret,optional"`
	NameSpace              string         `json:"namespace,optional"`
	SecurityToken          string         `json:"security_token,optional"`
	ConsumerConfig         ConsumerConfig `json:"consumer_config,optional"`
}

type ConsumerConfig struct {
	ConsumerGroup     string             `json:"consumer_group"`
	Handler           PullMessageHandler `json:"handler"`
	AwaitDuration     int64              `json:"await_duration,default=5"`
	PullBatchSize     int                `json:"pull_batch_size,default=32"`
	InvisibleDuration int64              `json:"invisible_duration,default=60"`
	TopicRelations    TopicRelation      `json:"topic_relations,optional"`
}

type TopicRelation struct {
	Topic          string `json:"topic"`
	Expression     string `json:"expression,default=*"`
	ExpressionType int    `json:"expression_type,default=1"`
}

type PullMessageHandler func(ctx context.Context, messages ...*golang.MessageView) (bool, error)
