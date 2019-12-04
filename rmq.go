package wlstmicro

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/streadway/amqp"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
	"github.com/xyzj/gopsu"
	"github.com/xyzj/gopsu/mq"
)

var (
	// mqProducer 生产者
	mqProducer *mq.Session
	// mqConsumer 消费者
	mqConsumer *mq.Session
	// gpsConsumer
	gpsConsumer     *mq.Session
	gpsRecvWaitLock sync.WaitGroup
)

// NewMQProducer NewRabbitmqProducer
func NewMQProducer() bool {
	if AppConf == nil {
		WriteError("SYS", "Configuration files should be loaded first")
		return false
	}
	rabbitConf.addr = AppConf.GetItemDefault("mq_addr", "127.0.0.1:5672", "mq服务地址,ip:port格式")
	rabbitConf.user = AppConf.GetItemDefault("mq_user", "arx7", "mq连接用户名")
	rabbitConf.pwd = gopsu.DecodeString(AppConf.GetItemDefault("mq_pwd", "WcELCNqP5dCpvMmMbKDdvgb", "mq连接密码"))
	rabbitConf.vhost = AppConf.GetItemDefault("mq_vhost", "", "mq虚拟域名")
	rabbitConf.exchange = AppConf.GetItemDefault("mq_exchange", "luwak_topic", "mq交换机名称")
	rabbitConf.enable, _ = strconv.ParseBool(AppConf.GetItemDefault("mq_enable", "true", "是否启用rabbitmq"))
	rabbitConf.usetls, _ = strconv.ParseBool(AppConf.GetItemDefault("mq_tls", "false", "是否使用证书连接rabbitmq服务"))
	if rabbitConf.usetls {
		rabbitConf.addr = strings.Replace(rabbitConf.addr, "5672", "5671", 1)
	}
	AppConf.Save()
	if !rabbitConf.enable {
		return false
	}
	mqProducer = mq.NewProducer(rabbitConf.exchange, fmt.Sprintf("amqp://%s:%s@%s/%s", rabbitConf.user, rabbitConf.pwd, rabbitConf.addr, rabbitConf.vhost), false)
	mqProducer.SetLogger(&StdLogger{
		Name:        "MQ",
		LogReplacer: strings.NewReplacer("[", "", "]", ""),
	})
	if rabbitConf.usetls {
		tc, err := gopsu.GetClientTLSConfig(RMQTLS.Cert, RMQTLS.Key, RMQTLS.ClientCA)
		if err != nil {
			WriteError("MQ", "RabbitMQ TLS Error: "+err.Error())
			return false
		}
		go mqProducer.StartTLS(tc)
	} else {
		go mqProducer.Start()
	}
	mqProducer.WaitReady(5)
	return true
}

// NewMQConsumer NewMQConsumer
func NewMQConsumer(svrName string) bool {
	if AppConf == nil {
		WriteError("SYS", "Configuration files should be loaded first")
		return false
	}
	rabbitConf.addr = AppConf.GetItemDefault("mq_addr", "127.0.0.1:5672", "mq服务地址,ip:port格式")
	rabbitConf.user = AppConf.GetItemDefault("mq_user", "arx7", "mq连接用户名")
	rabbitConf.pwd = gopsu.DecodeString(AppConf.GetItemDefault("mq_pwd", "WcELCNqP5dCpvMmMbKDdvgb", "mq连接密码"))
	rabbitConf.vhost = AppConf.GetItemDefault("mq_vhost", "", "mq虚拟域名")
	rabbitConf.exchange = AppConf.GetItemDefault("mq_exchange", "luwak_topic", "mq交换机名称")
	rabbitConf.queueRandom, _ = strconv.ParseBool(AppConf.GetItemDefault("mq_queue_random", "false", "随机队列名，true-用于独占模式，false-负载均衡（默认）"))
	rabbitConf.durable, _ = strconv.ParseBool(AppConf.GetItemDefault("mq_durable", "true", "队列是否持久化"))
	rabbitConf.autodel, _ = strconv.ParseBool(AppConf.GetItemDefault("mq_autodel", "true", "队列在未使用时是否删除"))
	rabbitConf.enable, _ = strconv.ParseBool(AppConf.GetItemDefault("mq_enable", "true", "是否启用rabbitmq"))

	if rabbitConf.usetls {
		rabbitConf.addr = strings.Replace(rabbitConf.addr, "5672", "5671", 1)
	}
	AppConf.Save()
	// 若不启用mq功能，则退出
	if !rabbitConf.enable {
		return false
	}
	rabbitConf.queue = rootPath + "_" + svrName
	if rabbitConf.queueRandom {
		rabbitConf.queue += "_" + gopsu.GetMD5(time.Now().Format("150405000"))
		rabbitConf.durable = false
		rabbitConf.autodel = true
	}
	mqConsumer = mq.NewConsumer(rabbitConf.exchange, fmt.Sprintf("amqp://%s:%s@%s/%s", rabbitConf.user, rabbitConf.pwd, rabbitConf.addr, rabbitConf.vhost), rabbitConf.queue, rabbitConf.durable, rabbitConf.autodel, false)
	mqConsumer.SetLogger(&StdLogger{
		Name: "MQ",
	})

	go mqConsumer.Start()
	mqConsumer.WaitReady(5)
	return true
}

// ProducerIsReady 返回ProducerIsReady可用状态
func ProducerIsReady() bool {
	if mqProducer != nil {
		return mqProducer.IsReady()
	}
	return false
}

// ConsumerIsReady 返回ProducerIsReady可用状态
func ConsumerIsReady() bool {
	if mqConsumer != nil {
		return mqConsumer.IsReady()
	}
	return false
}

// AppendRootPathRabbit 向rabbitmq的key追加头
func AppendRootPathRabbit(key string) string {
	if !strings.HasPrefix(key, rootPathMQ()) {
		return rootPathMQ() + key
	}
	return key
}

// BindRabbitMQ 绑定消费者key
func BindRabbitMQ(keys ...string) {
	if !ConsumerIsReady() {
		return
	}
	for _, v := range keys {
		if strings.TrimSpace(v) == "" {
			continue
		}
		mqConsumer.BindKey(AppendRootPathRabbit(v))
	}
}

// ReadRabbitMQ 接收消费者数据
func ReadRabbitMQ() (<-chan amqp.Delivery, error) {
	if !ConsumerIsReady() {
		return nil, fmt.Errorf("Consumer is not ready")
	}
	return mqConsumer.Recv()
}

// WriteRabbitMQ 写mq
func WriteRabbitMQ(key string, value []byte, expire time.Duration) {
	if !ConsumerIsReady() {
		return
	}
	key = AppendRootPathRabbit(key)
	mqProducer.SendCustom(&mq.RabbitMQData{
		RoutingKey: key,
		Data: &amqp.Publishing{
			ContentType:  "text/plain",
			DeliveryMode: amqp.Persistent,
			Expiration:   strconv.Itoa(int(expire.Nanoseconds() / 1000000)),
			Timestamp:    time.Now(),
			Body:         value,
		},
	})
	WriteInfo("MQ", "S:"+key+"|"+mq.FormatMQBody(value))
}

// PubEvent 事件id，状态，过滤器，用户名，详细，来源ip，额外数据
func PubEvent(eventid, status int, key, username, detail, from, jsdata string) {
	js, _ := sjson.Set("", "time", time.Now().Unix())
	js, _ = sjson.Set(js, "event_id", eventid)
	js, _ = sjson.Set(js, "user_name", username)
	js, _ = sjson.Set(js, "detail", detail)
	js, _ = sjson.Set(js, "from", from)
	js, _ = sjson.Set(js, "status", status)
	gjson.Parse(jsdata).ForEach(func(key, value gjson.Result) bool {
		js, _ = sjson.Set(js, key.Str, value.Value())
		return true
	})
	WriteRabbitMQ(key, []byte(js), time.Minute*10)
}
