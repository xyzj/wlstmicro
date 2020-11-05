package wmv2

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/xyzj/gopsu"
)

func machineCode() string {
	var b bytes.Buffer
	b.WriteString(fmt.Sprintf("%019d \\\n", runtime.NumCPU()))
	interfaces, err := net.Interfaces()
	if err != nil {
		panic("Error:" + err.Error())
	}
	for _, inter := range interfaces {
		n := strings.ToLower(inter.Name)
		if strings.Contains(n, "lo") || strings.HasPrefix(n, "v") || strings.HasPrefix(n, "t") || strings.HasPrefix(n, "d") || strings.HasPrefix(n, "is") {
			continue
		}
		b.WriteString(inter.HardwareAddr.String() + " \\\nend")
	}
	return base64.StdEncoding.EncodeToString(gopsu.DoZlibCompress(b.Bytes()))
}

func (fw *WMFrameWorkV2) checkMachine() {
	defer recover()
	mfile := filepath.Join(gopsu.GetExecDir(), ".firstrun")
	for _, v := range strings.Split(fw.versionInfo, "\n") {
		if strings.HasPrefix(v, "Version") {
			ver := gopsu.TrimString(strings.Split(v, ":")[1])
			if ver != "0.0.0" {
				ss := strings.Split(ver, ".")
				if ss[len(ss)-1] == "90060" { // need check machine
					b, err := ioutil.ReadFile(mfile)
					if err != nil {
						println("wrong machine.")
						os.Exit(21)
					}
					if gopsu.DecodeString(gopsu.TrimString(string(b))) != machineCode() {
						println("wrong machine.")
						os.Exit(21)
					}
				}
			}
		}
	}
}
