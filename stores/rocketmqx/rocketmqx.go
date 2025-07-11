package rocketmqx

import (
	"context"
	"github.com/apache/rocketmq-clients/golang"
	"github.com/apache/rocketmq-clients/golang/credentials"
	"github.com/og-game/glib/utils"
	"github.com/zeromicro/go-zero/core/logx"
	"os"
	"time"
)

type RocketMqx struct {
	config Config
}

func NewRocketMqx(config Config) *RocketMqx {
	_ = os.Setenv("mq.consoleAppender.enabled", utils.Ternary(config.ConsoleAppenderEnabled, "true", "false"))
	return &RocketMqx{config: config}
}

// 创建基础配置，减少重复代码
func (r *RocketMqx) createBaseConfig() *golang.Config {
	return &golang.Config{
		Endpoint:      r.config.Endpoint,
		NameSpace:     r.config.NameSpace,
		ConsumerGroup: r.config.ConsumerConfig.ConsumerGroup,
		Credentials: &credentials.SessionCredentials{
			AccessKey:     r.config.AccessKey,
			AccessSecret:  r.config.AccessSecret,
			SecurityToken: r.config.SecurityToken,
		},
	}
}

func (r *RocketMqx) NewProducer() (producer golang.Producer, err error) {
	producer, err = golang.NewProducer(r.createBaseConfig())
	if err != nil {
		logx.Errorf("NewProducer err: %s", err.Error())
		return
	}
	// 启动生产者
	if err = producer.Start(); err != nil {
		logx.Errorf("Start producer err: %s", err.Error())
		return
	}
	return
}

func (r *RocketMqx) NewConsumer(handler PullMessageHandler) (err error) {
	relations := map[string]*golang.FilterExpression{
		r.config.ConsumerConfig.TopicRelations.Topic: golang.NewFilterExpressionWithType(
			r.config.ConsumerConfig.TopicRelations.Expression,
			golang.FilterExpressionType(r.config.ConsumerConfig.TopicRelations.ExpressionType),
		),
	}

	simpleConsumer, err := golang.NewSimpleConsumer(
		r.createBaseConfig(),
		golang.WithAwaitDuration(time.Duration(r.config.ConsumerConfig.AwaitDuration)),
		golang.WithSubscriptionExpressions(relations),
	)
	if err != nil {
		logx.Errorf("初始化消费者失败，原因为：%s", err.Error())
		return
	}

	if err = simpleConsumer.Start(); err != nil {
		logx.Errorf("启动消费者失败，原因为：%s", err.Error())
		return
	}

	// 确保在函数退出时优雅停止消费者，并处理错误
	defer func() {
		if stopErr := simpleConsumer.GracefulStop(); stopErr != nil {
			logx.Errorf("优雅停止消费者失败，原因为：%s", stopErr.Error())
		}
	}()

	// 将消息处理逻辑提取到单独的函数中
	go r.processMessages(simpleConsumer, handler, r.config.ConsumerConfig.TopicRelations.Topic)

	return nil
}

// 处理消息的逻辑提取到单独的函数中
func (r *RocketMqx) processMessages(consumer golang.SimpleConsumer, handler PullMessageHandler, topic string) {
	for {
		ctx, cancel := context.WithTimeout(
			context.Background(),
			time.Duration(r.config.ConsumerConfig.AwaitDuration)*time.Second,
		)
		sleepTime := time.Duration(r.config.ConsumerConfig.AwaitDuration/2) * time.Second

		mvs, err := consumer.Receive(
			ctx,
			int32(r.config.ConsumerConfig.PullBatchSize),
			time.Duration(r.config.ConsumerConfig.InvisibleDuration)*time.Second,
		)

		if err != nil {
			logx.Errorf("拉取消息失败，topic:%s,原因为:%s", topic, err.Error())
			time.Sleep(sleepTime)
			continue
		}

		// 处理消息
		res, err := handler(ctx, mvs...)
		if err != nil {
			cancel()
			logx.Errorf("处理消息失败,topic:%s,原因为：%s", topic, err.Error())
			time.Sleep(sleepTime)
			continue
		}

		// 如果全部成功，确认消息
		if res {
			for _, mv := range mvs {
				if err = consumer.Ack(ctx, mv); err != nil {
					logx.Errorf("ack message failed, reason: %s, msgID:%s", err.Error(), mv.GetMessageId())
				}
			}
		}
	}
}
