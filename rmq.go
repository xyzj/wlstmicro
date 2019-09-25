package wlstmicro

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/xyzj/gopsu/mq"

	"github.com/streadway/amqp"
	"github.com/tidwall/sjson"
	"github.com/xyzj/gopsu"
)

var (
	// MQProducer 生产者
	MQProducer *mq.Session
	// MQConsumer 消费者
	MQConsumer *mq.Session
)

// NewMQProducer NewRabbitmqProducer
func NewMQProducer() {
	if AppConf == nil {
		WriteLog("SYS", "Configuration files should be loaded first", 40)
		return
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
	if !rabbitConf.enable {
		return
	}
	MQProducer = mq.NewProducer(rabbitConf.exchange, fmt.Sprintf("amqp://%s:%s@%s/%s", rabbitConf.user, rabbitConf.pwd, rabbitConf.addr, rabbitConf.vhost), false)
	if sysLog != nil {
		MQProducer.SetLogger(&sysLog.DefaultWriter, LogLevel)
	}
	if rabbitConf.usetls {
		tc, err := gopsu.GetClientTLSConfig(RMQTLS.Cert, RMQTLS.Key, RMQTLS.ClientCA)
		if err != nil {
			WriteLog("MQ", "RabbitMQ TLS Error: "+err.Error(), 40)
			return
		}
		go MQProducer.StartTLS(tc)
	} else {
		go MQProducer.Start()
	}
	activeRmq = true
}

// WriteRabbitMQ 写mq
func WriteRabbitMQ(key string, value []byte, expire time.Duration) {
	if !activeRmq {
		return
	}
	MQProducer.SendCustom(&mq.RabbitMQData{
		RoutingKey: key,
		Data: &amqp.Publishing{
			ContentType:  "text/plain",
			DeliveryMode: amqp.Persistent,
			Expiration:   strconv.Itoa(int(expire.Nanoseconds() / 1000000)),
			Timestamp:    time.Now(),
			Body:         value,
		},
	})
	if LogLevel >= 20 {
		WriteLog("MQ", "S:"+key+"|"+mq.FormatMQBody(value), 20)
	}
}

// PubEvent 事件id，状态，过滤器，用户名，详细，来源ip，额外数据
func PubEvent(eventid, status int, key, username, detail, from, jsdata string) {
	js, _ := sjson.Set("", "time", time.Now().Unix())
	js, _ = sjson.Set(js, "event_id", eventid)
	js, _ = sjson.Set(js, "username", username)
	js, _ = sjson.Set(js, "detail", detail)
	js, _ = sjson.Set(js, "from", from)
	js, _ = sjson.Set(js, "status", status)
	WriteRabbitMQ(key, []byte(js), time.Minute*10)
}

// ProducerIsReady 返回ProducerIsReady可用状态
func ProducerIsReady() bool {
	return MQProducer != nil
}

// NewMQConsumer NewMQConsumer
func NewMQConsumer() {
	if AppConf == nil {
		WriteLog("SYS", "Configuration files should be loaded first", 40)
		return
	}
	rabbitConf.addr = AppConf.GetItemDefault("mq_addr", "127.0.0.1:5672", "mq服务地址,ip:port格式")
	rabbitConf.user = AppConf.GetItemDefault("mq_user", "arx7", "mq连接用户名")
	rabbitConf.pwd = gopsu.DecodeString(AppConf.GetItemDefault("mq_pwd", "WcELCNqP5dCpvMmMbKDdvgb", "mq连接密码"))
	rabbitConf.vhost = AppConf.GetItemDefault("mq_vhost", "", "mq虚拟域名")
	rabbitConf.exchange = AppConf.GetItemDefault("mq_exchange", "luwak_topic", "mq交换机名称")
	rabbitConf.queue = AppConf.GetItemDefault("mq_queue", "abc", "mq队列名称")
	rabbitConf.durable, _ = strconv.ParseBool(AppConf.GetItemDefault("mq_durable", "true", "队列是否持久化"))
	rabbitConf.autodel, _ = strconv.ParseBool(AppConf.GetItemDefault("mq_autodel", "true", "队列在未使用时是否删除"))
	rabbitConf.enable, _ = strconv.ParseBool(AppConf.GetItemDefault("mq_enable", "true", "是否启用rabbitmq"))
	if rabbitConf.usetls {
		rabbitConf.addr = strings.Replace(rabbitConf.addr, "5672", "5671", 1)
	}
	if !rabbitConf.enable {
		return
	}
	MQConsumer = mq.NewConsumer(rabbitConf.exchange, fmt.Sprintf("amqp://%s:%s@%s/%s", rabbitConf.user, rabbitConf.pwd, rabbitConf.addr, rabbitConf.vhost), rabbitConf.queue, rabbitConf.durable, rabbitConf.autodel, false)
	if sysLog != nil {
		MQConsumer.SetLogger(&sysLog.DefaultWriter, LogLevel)
	}
	go MQConsumer.Start()
	activeRmq = true
}
