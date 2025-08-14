package rocketmqx

import (
	"context"
	"github.com/apache/rocketmq-clients/golang/v5"
	"github.com/apache/rocketmq-clients/golang/v5/credentials"
	v2 "github.com/apache/rocketmq-clients/golang/v5/protocol/v2"
	"github.com/og-game/glib/utils"
	"github.com/zeromicro/go-zero/core/logx"
	"os"
	"strings"
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
		golang.WithAwaitDuration(time.Duration(r.config.ConsumerConfig.AwaitDuration)*time.Second),
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

	// 将消息处理逻辑提取到单独的函数中
	go r.processMessages(simpleConsumer, handler, r.config.ConsumerConfig.TopicRelations.Topic)

	return nil
}

// 处理消息的逻辑提取到单独的函数中
func (r *RocketMqx) processMessages(consumer golang.SimpleConsumer, handler PullMessageHandler, topic string) {
	for {
		// 1. 拉取消息 - Receive超时设置为 AwaitDuration + 5秒buffer
		receiveCtx, receiveCancel := context.WithTimeout(
			context.Background(),
			time.Duration(r.config.ConsumerConfig.AwaitDuration+5)*time.Second,
		)
		mvs, err := consumer.Receive(
			receiveCtx,
			int32(r.config.ConsumerConfig.PullBatchSize),
			time.Duration(r.config.ConsumerConfig.InvisibleDuration)*time.Second,
		)
		receiveCancel()

		// 2. 处理拉取错误
		if err != nil {
			if strings.Contains(err.Error(), v2.Code_name[int32(v2.Code_MESSAGE_NOT_FOUND)]) {
				// 无消息时短暂休眠
				time.Sleep(time.Duration(r.config.ConsumerConfig.AwaitDuration/3) * time.Second)
				continue
			}
			logx.Errorf("拉取消息失败，topic:%s,原因为:%s", topic, err.Error())
			time.Sleep(time.Duration(r.config.ConsumerConfig.AwaitDuration/2) * time.Second)
			continue
		}

		// 3. 处理消息 - 使用 InvisibleDuration 作为处理超时
		handlerCtx, handlerCancel := context.WithTimeout(
			context.Background(),
			time.Duration(r.config.ConsumerConfig.InvisibleDuration)*time.Second,
		)

		res, err := handler(handlerCtx, mvs...)
		handlerCancel()

		// 4. ACK确认
		if res && err == nil {
			// ACK使用独立的短超时context
			ackCtx, ackCancel := context.WithTimeout(context.Background(), 5*time.Second)
			for _, mv := range mvs {
				if ackErr := consumer.Ack(ackCtx, mv); ackErr != nil {
					logx.Errorf("ack message failed, reason: %s, msgID:%s", ackErr.Error(), mv.GetMessageId())
				}
			}
			ackCancel()
		} else if err != nil {
			logx.Errorf("处理消息失败,topic:%s,原因为：%s", topic, err.Error())
			// 处理失败短暂休眠
			time.Sleep(time.Second)
		}
	}
}
