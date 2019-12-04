package wlstmicro

import (
	"fmt"
	"math"
	"os/exec"
	"time"

	"github.com/tidwall/gjson"
	"github.com/xyzj/gopsu"
	"github.com/xyzj/gopsu/mq"
)

// 启用gps校时
func newGPSConsumer(svrName string) {
	rabbitConf.addr = AppConf.GetItemDefault("mq_addr", "127.0.0.1:5672", "mq服务地址,ip:port格式")
	rabbitConf.user = AppConf.GetItemDefault("mq_user", "arx7", "mq连接用户名")
	rabbitConf.pwd = gopsu.DecodeString(AppConf.GetItemDefault("mq_pwd", "WcELCNqP5dCpvMmMbKDdvgb", "mq连接密码"))
	rabbitConf.vhost = AppConf.GetItemDefault("mq_vhost", "", "mq虚拟域名")
	rabbitConf.exchange = AppConf.GetItemDefault("mq_exchange", "luwak_topic", "mq交换机名称")
	AppConf.Save()
	queue := rootPath + "_" + svrName + "_gps_" + gopsu.GetMD5(time.Now().Format("150405000"))
	durable := false
	autodel := true
	gpsConsumer = mq.NewConsumer(rabbitConf.exchange, fmt.Sprintf("amqp://%s:%s@%s/%s", rabbitConf.user, rabbitConf.pwd, rabbitConf.addr, rabbitConf.vhost), queue, durable, autodel, false)
	gpsConsumer.SetLogger(&StdLogger{
		Name: "MQGPS",
	})

	go gpsConsumer.Start()
	gpsConsumer.WaitReady(5)

	gpsConsumer.BindKey(AppendRootPathRabbit("gps.serlreader.#"))
	go gpsRecv()
	go handerGPSRecv()
}

func handerGPSRecv() {
	for {
		time.Sleep(time.Second)
		gpsRecvWaitLock.Wait()
		go gpsRecv()
	}
}
func gpsRecv() {
	defer func() {
		if err := recover(); err != nil {
			WriteError("MQGPS", "Rcv Crash: "+err.(error).Error())
		}
		gpsRecvWaitLock.Done()
	}()
	gpsRecvWaitLock.Add(1)
	rcvMQ, err := gpsConsumer.Recv()
	if err != nil {
		WriteError("MQGPS", "Rcv Err: "+err.Error())
		return
	}
	for d := range rcvMQ {
		WriteDebug("MQGPS", "R:"+d.RoutingKey+"|"+string(d.Body))
		gpsData := gjson.ParseBytes(d.Body)
		if math.Abs(float64(gpsData.Get("cache_time").Int()-time.Now().Unix())) < 30 {
			switch rabbitConf.gpsTiming {
			case 0: // 不校时，不存在这个情况，姑且写在这里
			case 1: // 50～900s范围校时
				if math.Abs(float64(time.Now().Unix()-gpsData.Get("gps_time").Int())) > 50 && math.Abs(float64(time.Now().Unix()-gpsData.Get("gps_time").Int())) < 900 {
					modifyTime(gpsData.Get("gps_time").Int())
				}
			case 2: // 强制校时
				modifyTime(gpsData.Get("gps_time").Int())
			}
		}
	}
}
func modifyTime(t int64) {
	gd := time.Unix(t, 5)
	year, month, day := gd.Date()
	hour, minute, second := gd.Clock()
	WriteSystem("MQGPS", "Modify system time from "+time.Now().Format(gopsu.LongTimeFormat)+" to "+gd.Format(gopsu.LongTimeFormat))
	if gopsu.OSNAME == "windows" {
		cmd := exec.Command("date", fmt.Sprintf("%04d-%02d-%02d", year, month, day))
		cmd.Run()
		cmd = exec.Command("time", fmt.Sprintf("%02d:%02d:%02d", hour, minute, second))
		cmd.Run()
	} else {
		cmd := exec.Command("date", fmt.Sprintf("%02d%02d%02d%02d%02d.%02d", month, day, hour, minute, year, second))
		cmd.Run()
		cmd = exec.Command("hwclock -w")
		cmd.Run()
	}
}
