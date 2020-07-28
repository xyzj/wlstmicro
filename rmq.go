package wlstmicro

import (
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"
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
	// 消费者监控锁
	mqRecvWaitLock sync.WaitGroup
	rabbitConf     = &rabbitConfigure{}
)

// rabbitmq配置
type rabbitConfigure struct {
	forshow string
	// rmq服务地址
	addr string
	// 登录用户名
	user string
	// 登录密码
	pwd string
	// 虚拟域名
	vhost string
	// 交换机名称
	exchange string
	// 队列名称
	queue string
	// 是否使用随机队列名
	queueRandom bool
	// 队列是否持久化
	durable bool
	// 队列是否在未使用时自动删除
	autodel bool
	// 是否启用tls
	usetls bool
	// 是否启用rmq
	enable bool
	// 启用gps校时,0-不启用，1-启用（30～900s内进行矫正），2-强制对时
	gpsTiming int64
}

func (conf *rabbitConfigure) show() string {
	conf.forshow, _ = sjson.Set("", "addr", rabbitConf.addr)
	conf.forshow, _ = sjson.Set(conf.forshow, "user", CWorker.Encrypt(rabbitConf.user))
	conf.forshow, _ = sjson.Set(conf.forshow, "pwd", CWorker.Encrypt(rabbitConf.pwd))
	conf.forshow, _ = sjson.Set(conf.forshow, "vhost", rabbitConf.vhost)
	conf.forshow, _ = sjson.Set(conf.forshow, "exchange", rabbitConf.exchange)
	conf.forshow, _ = sjson.Set(conf.forshow, "use_tls", rabbitConf.usetls)
	conf.forshow, _ = sjson.Set(conf.forshow, "root_path", rootPath)
	return conf.forshow
}

// NewMQProducer NewRabbitmqProducer
func NewMQProducer() bool {
	if AppConf == nil {
		WriteError("SYS", "Configuration files should be loaded first")
		return false
	}
	rabbitConf.addr = AppConf.GetItemDefault("mq_addr", "127.0.0.1:5671", "mq服务地址,ip:port格式")
	rabbitConf.user = AppConf.GetItemDefault("mq_user", "arx7", "mq连接用户名")
	rabbitConf.pwd = gopsu.DecodeString(AppConf.GetItemDefault("mq_pwd", "WcELCNqP5dCpvMmMbKDdvgb", "mq连接密码"))
	rabbitConf.vhost = AppConf.GetItemDefault("mq_vhost", "", "mq虚拟域名")
	rabbitConf.exchange = AppConf.GetItemDefault("mq_exchange", "luwak_topic", "mq交换机名称")
	rabbitConf.enable, _ = strconv.ParseBool(AppConf.GetItemDefault("mq_enable", "true", "是否启用rabbitmq"))
	rabbitConf.usetls, _ = strconv.ParseBool(AppConf.GetItemDefault("mq_tls", "true", "是否使用证书连接rabbitmq服务"))
	rmqProtocol := "amqps"
	if !rabbitConf.usetls {
		rabbitConf.addr = strings.Replace(rabbitConf.addr, "5671", "5672", 1)
		rmqProtocol = "amqp"
	}
	AppConf.Save()
	rabbitConf.show()
	if !rabbitConf.enable {
		return false
	}
	mqProducer = mq.NewProducer(rabbitConf.exchange, fmt.Sprintf("%s://%s:%s@%s/%s", rmqProtocol, rabbitConf.user, rabbitConf.pwd, rabbitConf.addr, rabbitConf.vhost), false)
	mqProducer.SetLogger(&StdLogger{
		Name:        "MQ",
		LogReplacer: strings.NewReplacer("[", "", "]", ""),
	})
	if rabbitConf.usetls {
		// tc, err := gopsu.GetClientTLSConfig(RMQTLS.Cert, RMQTLS.Key, RMQTLS.ClientCA)
		// if err != nil {
		// 	WriteError("MQ", "RabbitMQ Producer TLS Error: "+err.Error())
		// 	return false
		// }
		return mqProducer.StartTLS(&tls.Config{InsecureSkipVerify: true})
	}
	return mqProducer.Start()
}

// NewMQConsumer NewMQConsumer
func NewMQConsumer(svrName string) bool {
	if AppConf == nil {
		WriteError("SYS", "Configuration files should be loaded first")
		return false
	}
	rabbitConf.addr = AppConf.GetItemDefault("mq_addr", "127.0.0.1:5671", "mq服务地址,ip:port格式")
	rabbitConf.user = AppConf.GetItemDefault("mq_user", "arx7", "mq连接用户名")
	rabbitConf.pwd = gopsu.DecodeString(AppConf.GetItemDefault("mq_pwd", "WcELCNqP5dCpvMmMbKDdvgb", "mq连接密码"))
	rabbitConf.vhost = AppConf.GetItemDefault("mq_vhost", "", "mq虚拟域名")
	rabbitConf.exchange = AppConf.GetItemDefault("mq_exchange", "luwak_topic", "mq交换机名称")
	rabbitConf.queueRandom, _ = strconv.ParseBool(AppConf.GetItemDefault("mq_queue_random", "false", "随机队列名，true-用于独占模式，false-负载均衡（默认）"))
	rabbitConf.durable, _ = strconv.ParseBool(AppConf.GetItemDefault("mq_durable", "false", "队列是否持久化"))
	rabbitConf.autodel, _ = strconv.ParseBool(AppConf.GetItemDefault("mq_autodel", "true", "队列在未使用时是否删除"))
	rabbitConf.enable, _ = strconv.ParseBool(AppConf.GetItemDefault("mq_enable", "true", "是否启用rabbitmq"))
	rabbitConf.usetls, _ = strconv.ParseBool(AppConf.GetItemDefault("mq_tls", "true", "是否使用证书连接rabbitmq服务"))
	rmqProtocol := "amqps"
	if !rabbitConf.usetls {
		rabbitConf.addr = strings.Replace(rabbitConf.addr, "5671", "5672", 1)
		rmqProtocol = "amqp"
	}
	AppConf.Save()

	rabbitConf.show()
	// 若不启用mq功能，则退出
	if !rabbitConf.enable {
		return false
	}
	rabbitConf.queue = rootPath + "_" + svrName
	if rabbitConf.queueRandom {
		rabbitConf.queue += "_" + MD5Worker.Hash([]byte(time.Now().Format("150405000")))
		rabbitConf.durable = false
		rabbitConf.autodel = true
	}
	mqConsumer = mq.NewConsumer(rabbitConf.exchange,
		fmt.Sprintf("%s://%s:%s@%s/%s", rmqProtocol, rabbitConf.user, rabbitConf.pwd, rabbitConf.addr, rabbitConf.vhost),
		rabbitConf.queue,
		rabbitConf.durable,
		rabbitConf.autodel,
		false)
	mqConsumer.SetLogger(&StdLogger{
		Name: "MQ",
	})
	if rabbitConf.usetls {
		// tc, err := gopsu.GetClientTLSConfig(RMQTLS.Cert, RMQTLS.Key, RMQTLS.ClientCA)
		// if err != nil {
		// 	WriteError("MQ", "RabbitMQ Consumer TLS Error: "+err.Error())
		// 	return false
		// }
		return mqConsumer.StartTLS(&tls.Config{InsecureSkipVerify: true})
	}
	return mqConsumer.Start()
}

// RecvRabbitMQ 接收消息
// f: 消息处理方法，key为消息过滤器，body为消息体
func RecvRabbitMQ(f func(key string, body []byte)) {
	var mqRecvWaitLock sync.WaitGroup
RECV:
	mqRecvWaitLock.Add(1)
	go func() {
		defer func() {
			if err := recover(); err != nil {
				WriteError("MQ", "Consumer core crash: "+errors.WithStack(err.(error)).Error())
			}
			mqRecvWaitLock.Done()
		}()
		rcvMQ, err := mqConsumer.Recv()
		if err != nil {
			WriteError("MQ", "Rcv Err: "+err.Error())
			return
		}
		for d := range rcvMQ {
			if gjson.ValidBytes(d.Body) {
				WriteDebug("MQ", "Debug-R:"+rabbitConf.addr+"|"+d.RoutingKey+"|"+string(d.Body))
			} else {
				WriteDebug("MQ", "Debug-R:"+rabbitConf.addr+"|"+d.RoutingKey+"|"+base64.StdEncoding.EncodeToString(d.Body))
			}
			f(d.RoutingKey, d.Body)
		}
	}()

	mqRecvWaitLock.Wait()
	time.Sleep(time.Second * 15)
	goto RECV
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
	kk := make([]string, 0)
	for _, v := range keys {
		if strings.TrimSpace(v) == "" {
			continue
		}
		kk = append(kk, AppendRootPathRabbit(v))
	}
	mqConsumer.BindKey(kk...)
}

// UnBindRabbitMQ 解除绑定消费者key
func UnBindRabbitMQ(keys ...string) {
	kk := make([]string, 0)
	for _, v := range keys {
		if strings.TrimSpace(v) == "" {
			continue
		}
		kk = append(kk, AppendRootPathRabbit(v))
	}
	mqConsumer.UnBindKey(kk...)
}

// ReadRabbitMQ 获得消费者消息通道 (Obsolete,please just call RecvRabbitMQ(func(key string,body []body)))
func ReadRabbitMQ() (<-chan amqp.Delivery, error) {
	if !ConsumerIsReady() {
		return nil, fmt.Errorf("Consumer is not ready")
	}
	return mqConsumer.Recv()
}

// WriteRabbitMQ 写mq
func WriteRabbitMQ(key string, value []byte, expire time.Duration) {
	if !ProducerIsReady() {
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
	// WriteInfo("MQ", "S:"+key+"|"+mq.FormatMQBody(value))
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

// ClearQueue 清空队列
func ClearQueue() {
	if !ConsumerIsReady() {
		return
	}
	mqConsumer.ClearQueue()
}

// ViewRabbitMQConfig 查看rabbitmq配置,返回json字符串
func ViewRabbitMQConfig() string {
	return rabbitConf.forshow
}
