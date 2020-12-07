package wmv2

import (
	"context"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/mohae/deepcopy"

	"github.com/pkg/errors"
	"github.com/xyzj/gopsu"
	msgctl "github.com/xyzj/proto/msgjk"
)

var (
	idx uint64 // socket id
)

type tcpConfigure struct {
	tcpClientsManager *gopsu.Queue //= gopsu.NewQueue()                  // socket实例池
	tcpClients        sync.Map     //= make(map[uint64]*TCPBase)         // 有效的socket实例字典
	onlineSocks       string       // 在线设备的json字符串
	readTimeout       int64        // 读取超时
	idleTimeout       int64        // 发送闲置超时
	mqFlag            string       // mq发送标识
	matchOne          bool         // 是否只匹配一个
	filterIP          bool         // 过滤ip，仅允许合法ip连接，从redis获取
}

// TCPBase tcp 模块基础接口
type TCPBase interface {
	// New 初始化内部变量，传递读取超时和发送超时，毫秒
	New()
	// ID 返回关键id
	ID() uint64
	// RemoteAddr 返回远端地址
	RemoteAddr() string
	// Connect 连接设置
	Connect(*net.TCPConn) error
	// Disconnect 断开连接，需填原因
	Disconnect(string)
	// Clean 清理内部变量
	Clean()
	// Send 发送方法
	Send(context.Context)
	// Recv 接收方法
	Recv()
	// Put 设置发送内容
	Put(interface{}) error
	// 检查状态
	StatusCheck() string
}

type illegalIP struct {
	ip     []string
	locker sync.RWMutex
}

func (i *illegalIP) Set(s string) {
	i.locker.Lock()
	defer i.locker.Unlock()
	i.ip = strings.Split(s, ",")
}
func (i *illegalIP) Check(s string) bool {
	for _, v := range i.ip {
		if v == s {
			return true
		}
	}
	return false
}

func (fw *WMFrameWorkV2) tcpHandler() {
	var locker sync.WaitGroup
	fw.tcpCtl.tcpClientsManager = gopsu.NewQueue()
RUN:
	go func() {
		defer func() {
			if err := recover(); err != nil {
				fw.WriteError("TCP", "Core crash:"+err.(error).Error())
			}
			locker.Done()
		}()
		locker.Add(1)
		tickCheckTCP := time.Tick(time.Second * 15)
		checkCount := 0
		for {
			select {
			case msg := <-fw.chanTCPWorker: // 检查发送数据
				fw.tcpCtl.tcpClients.Range(func(key interface{}, value interface{}) bool {
					if value.(TCPBase).Put(msg) != nil { // 不匹配目标，继续
						return true
					}
					// 已匹配，判断是否继续匹配
					if fw.tcpCtl.matchOne { // 仅匹配一个
						return false
					}
					// 继续找下一个
					return true
				})
			case <-tickCheckTCP: // 检查状态
				var sock = 0
				msg := &msgctl.MsgWithCtrl{
					Head: &msgctl.Head{
						Mod:  1,
						Src:  1,
						Ver:  1,
						Tver: 1,
						Ret:  1,
						Cmd:  "wlst.sys.onlineinfo",
						Tra:  1,
						Dst:  2,
						Dt:   time.Now().Unix(),
					},
					Args: &msgctl.Args{
						Port: int32(*tcpPort),
					},
					Syscmds: &msgctl.SysCommands{
						Port: int32(*tcpPort),
					},
				}
				fw.tcpCtl.tcpClients.Range(func(key interface{}, value interface{}) bool {
					v := value.(TCPBase)
					if a := v.StatusCheck(); a != "" {
						sock++
						msginfo := &msgctl.SysCommands_OnlineInfo{}
						msginfo.Unmarshal([]byte(a))
						if msginfo.String() != "" {
							msg.Syscmds.OnlineInfo = append(msg.Syscmds.OnlineInfo, msginfo)
						}
					}
					return true
				})
				b, _ := msg.Marshal()
				fw.WriteRabbitMQ(fmt.Sprintf("devonline.%s.%s", fw.serverName, fw.tcpCtl.mqFlag), b, time.Second*15, &msgctl.MsgWithCtrl{})
				fw.tcpCtl.onlineSocks = string(gopsu.PB2Json(msg))
				fw.WriteRedis(fmt.Sprintf("devonline/%s/%s", fw.serverName, fw.tcpCtl.mqFlag), fw.tcpCtl.onlineSocks, time.Minute)
				checkCount++
				if checkCount >= 4 {
					checkCount = 0
					fw.WriteSystem("TCP", fmt.Sprintf("(%d) ActiveClients:%d, ClientsPool:%d", *tcpPort, sock, fw.tcpCtl.tcpClientsManager.Len()))
				}
			}
		}
	}()
	time.Sleep(time.Second)
	locker.Wait()
	goto RUN
}

func (fw *WMFrameWorkV2) newTCPService(t TCPBase) {
	fw.tcpCtl.mqFlag = fw.wmConf.GetItemDefault("mq_flag", "0", "设备上下行mq消息，额外区分标识")
	fw.tcpCtl.matchOne, _ = strconv.ParseBool(fw.wmConf.GetItemDefault("match_one", "true", "发送TCP命令时是否只匹配一个目标socket"))
	fw.tcpCtl.filterIP, _ = strconv.ParseBool(fw.wmConf.GetItemDefault("filter_ip", "false", "仅允许合法ip连接"))
	fw.wmConf.Save()
	// 检查端口
	if *tcpPort < 1000 || *tcpPort > 65535 {
		fw.WriteError("TCP", "Forbidden port range")
		return
	}
	// 处理合法ip
	var ipList = &illegalIP{}
	if fw.tcpCtl.filterIP { // 查询合法ip
		go func() {
			defer func() { recover() }()
			for {
				if z, err := fw.ReadRedis("legalips/dataparser-wlst"); err == nil {
					ipList.Set(z)
				}
				time.Sleep(time.Minute)
			}
		}()
	}

	go fw.tcpHandler()

	listener, ex := net.ListenTCP("tcp", &net.TCPAddr{IP: net.ParseIP(""), Port: *tcpPort, Zone: ""})
	if ex != nil {
		fw.WriteError("TCP", fmt.Sprintf("%s", ex.Error()))
		return
	}
	fw.WriteSystem("TCP", fmt.Sprintf("Success bind on port %d", *tcpPort))
	defer func() {
		if ex := recover(); ex != nil {
			fw.WriteError("TCP", fmt.Sprintf("TCP listener(%d) crash, NEED RESTART: %+v", *tcpPort, errors.WithStack(ex.(error))))
		}
		listener.Close()
	}()
	for {
		conn, ex := listener.AcceptTCP()
		if ex != nil {
			time.Sleep(10 * time.Millisecond)
			continue
		}
		// 检查合法ip
		if fw.tcpCtl.filterIP {
			if !ipList.Check(strings.Split(conn.RemoteAddr().String(), ":")[0]) {
				conn.Close()
				fw.WriteWarning("TCP "+conn.RemoteAddr().String(), fmt.Sprintf("Illegal connection to %d, KICK OUT", *tcpPort))
				continue
			}
		}
		fw.WriteWarning("TCP "+conn.RemoteAddr().String(), fmt.Sprintf("Connect to %d", *tcpPort))
		var cli TCPBase
		if a := fw.tcpCtl.tcpClientsManager.Get(); a != nil {
			cli = a.(TCPBase)
		} else { // 连接池为空，创建新实例
			cli = deepcopy.Copy(t).(TCPBase)
			cli.New()
		}

		go func(cli TCPBase, conn *net.TCPConn) {
			var sockLocker sync.WaitGroup
			var ctx, cancel = context.WithCancel(context.TODO())
			defer func() {
				if err := recover(); err != nil {
					fw.WriteError("TCP", err.(error).Error())
				}
				cli.Clean()
				fw.tcpCtl.tcpClients.Delete(cli.ID())
				fw.tcpCtl.tcpClientsManager.Put(cli)
				cancel()
			}()
			cli.Connect(conn)
			fw.tcpCtl.tcpClients.Store(cli.ID(), cli)

			// 发送线程
			go func(cli TCPBase, ctx context.Context) {
				defer func() {
					if err := recover(); err != nil {
						cli.Disconnect("Snd goroutine crash:" + errors.WithStack(err.(error)).Error())
					}
					sockLocker.Done()
				}()
				sockLocker.Add(1)
				cli.Send(ctx)
			}(cli, ctx)
			// 接收线程
			go func(cli TCPBase) {
				defer func() {
					if err := recover(); err != nil {
						cli.Disconnect("Rcv goroutine crash:" + errors.WithStack(err.(error)).Error())
					}
					cancel()
					sockLocker.Done()
				}()
				sockLocker.Add(1)
				cli.Recv()
			}(cli)
			time.Sleep(time.Second)
			sockLocker.Wait()
		}(cli, conn)
	}
}

type tcpEcho struct {
	conn       *net.TCPConn
	cliID      uint64
	closed     bool
	sendQueue  *gopsu.Queue
	readBuffer []byte
	logHead    string
}

func (t *tcpEcho) New() {
	t.cliID = atomic.AddUint64(&idx, 1)
	t.sendQueue = gopsu.NewQueue()
	t.readBuffer = make([]byte, 8192)
}
func (t *tcpEcho) Clean() {
	t.cliID = 0
	t.sendQueue.Clear()
	t.conn.Close()
}
func (t *tcpEcho) Put(d interface{}) error {
	t.sendQueue.Put(d)
	return nil
}
func (t *tcpEcho) Send(context.Context) {
	for {
		if t.closed {
			break
		}
		if a := t.sendQueue.Get(); a != nil {
			t.conn.Write(a.([]byte))
			println("Snd:" + string(a.([]byte)))
		}
		time.Sleep(time.Millisecond * 200)
	}
}
func (t *tcpEcho) Recv() {
	for {
		if t.closed {
			break
		}
		if ex := t.conn.SetReadDeadline(time.Now().Add(time.Second * 60)); ex != nil {
			t.Disconnect("set read timeout error:" + ex.Error())
			break
		}
		n, ex := t.conn.Read(t.readBuffer)
		if ex != nil {
			if ex == io.EOF {
				t.Disconnect("remote close:" + ex.Error())
			} else {
				t.Disconnect("RcvErr:" + ex.Error())
			}
			break
		}
		if n > 0 {
			d := t.readBuffer[:n]
			println("Rcv:" + string(d))
			t.Put(d)
		}
	}
}
func (t *tcpEcho) Connect(conn *net.TCPConn) error {
	t.conn = conn
	t.logHead = fmt.Sprintf("(%s)-%d ", t.conn.RemoteAddr().String(), t.cliID)
	return nil
}
func (t *tcpEcho) Disconnect(why string) {
	t.conn.Close()
	t.closed = true
	println("client close: " + why)
}
func (t *tcpEcho) ID() uint64 {
	return t.cliID
}
func (t *tcpEcho) StatusCheck() string {
	return "ok"
}
func (t *tcpEcho) RemoteAddr() string {
	return t.conn.RemoteAddr().String()
}
