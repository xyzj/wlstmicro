package wmv2

import (
	"crypto/tls"
	"fmt"
	"math"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/tidwall/gjson"
	"github.com/xyzj/gopsu"
	"github.com/xyzj/gopsu/mq"
)

// 启用gps校时
func (fw *WMFrameWorkV2) newGPSConsumer() {
	fw.loadMQConfig()
	queue := fw.rootPath + "_" + fw.serverName + "_gps_" + MD5Worker.Hash([]byte(time.Now().Format("150405000")))
	durable := false
	autodel := true
	fw.rmqCtl.gpsConsumer = mq.NewConsumer(fw.rmqCtl.exchange, fmt.Sprintf("%s://%s:%s@%s/%s", fw.rmqCtl.protocol, fw.rmqCtl.user, fw.rmqCtl.pwd, fw.rmqCtl.addr, fw.rmqCtl.vhost), queue, durable, autodel, false)
	fw.rmqCtl.gpsConsumer.SetLogger(&StdLogger{
		Name:        "MQGPS",
		LogReplacer: strings.NewReplacer("[", "", "]", ""),
		LogWriter:   fw.wmLog,
	})

	if fw.rmqCtl.usetls {
		fw.rmqCtl.gpsConsumer.StartTLS(&tls.Config{InsecureSkipVerify: true})
	} else {
		fw.rmqCtl.gpsConsumer.Start()
	}

	fw.rmqCtl.gpsConsumer.BindKey(fw.AppendRootPathRabbit("gps.serlreader.#"))
	go fw.gpsRecv()
}

func (fw *WMFrameWorkV2) gpsRecv() {
	var gpsRecvWaitLock sync.WaitGroup
RECV:
	gpsRecvWaitLock.Add(1)
	go func() {
		defer func() {
			if err := recover(); err != nil {
				fw.WriteError("MQGPS", "Rcv Crash: "+errors.WithStack(err.(error)).Error())
			}
			gpsRecvWaitLock.Done()
		}()
		rcvMQ, err := fw.rmqCtl.gpsConsumer.Recv()
		if err != nil {
			fw.WriteError("MQGPS", "Rcv Err: "+err.Error())
			return
		}
		for d := range rcvMQ {
			fw.WriteDebug("MQGPS", "Debug-R:"+d.RoutingKey+"|"+string(d.Body))
			gpsData := gjson.ParseBytes(d.Body)
			if math.Abs(float64(gpsData.Get("cache_time").Int()-time.Now().Unix())) < 30 {
				switch fw.gpsTimer {
				case 0: // 不校时，不存在这个情况，姑且写在这里
				case 1: // 50～900s范围校时
					if math.Abs(float64(time.Now().Unix()-gpsData.Get("gps_time").Int())) > 50 && math.Abs(float64(time.Now().Unix()-gpsData.Get("gps_time").Int())) < 900 {
						fw.modifyTime(gpsData.Get("gps_time").Int())
					}
				case 2: // 强制校时
					fw.modifyTime(gpsData.Get("gps_time").Int())
				}
			}
		}
	}()

	gpsRecvWaitLock.Wait()
	time.Sleep(time.Second * 15)
	goto RECV
}

func (fw *WMFrameWorkV2) modifyTime(t int64) {
	gd := time.Unix(t, 5)
	year, month, day := gd.Date()
	hour, minute, second := gd.Clock()
	fw.WriteSystem("MQGPS", "Modify system time from "+time.Now().Format(gopsu.LongTimeFormat)+" to "+gd.Format(gopsu.LongTimeFormat))
	if gopsu.OSNAME == "windows" {
		cmd := exec.Command("date", fmt.Sprintf("%04d-%02d-%02d", year, month, day))
		cmd.Run()
		cmd = exec.Command("time", fmt.Sprintf("%02d:%02d:%02d", hour, minute, second))
		cmd.Run()
	} else {
		cmd := exec.Command("date", fmt.Sprintf("%02d%02d%02d%02d%02d.%02d", month, day, hour, minute, year, second))
		cmd.Run()
		cmd = exec.Command("hwclock", " -w")
		cmd.Run()
	}
}
