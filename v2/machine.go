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

	"github.com/tidwall/gjson"

	"github.com/xyzj/gopsu"
)

func machineCode() string {
	var b bytes.Buffer
	b.WriteString(fmt.Sprintf("%03d", runtime.NumCPU()))
	interfaces, err := net.Interfaces()
	if err != nil {
		panic("Error: try run with administrator rights")
	}
	for _, inter := range interfaces {
		n := strings.ToLower(inter.Name)
		if strings.Contains(n, "lo") || strings.HasPrefix(n, "v") || strings.HasPrefix(n, "t") || strings.HasPrefix(n, "d") || strings.HasPrefix(n, "is") {
			continue
		}
		b.WriteString(inter.HardwareAddr.String())
	}
	return strings.ReplaceAll(base64.StdEncoding.EncodeToString(gopsu.CompressData(b.Bytes(), gopsu.ArchiveGZip))[3:], "=", "")
}

func (fw *WMFrameWorkV2) checkMachine() {
	defer recover()
	mfile := filepath.Join(gopsu.GetExecDir(), ".firstrun")
	ver := gjson.Parse(fw.verJSON).Get("version").String()
	if ver != "0.0.0" {
		ss := strings.Split(ver, ".")
		if ss[len(ss)-1] == "90060" { // need check machine
			b, err := ioutil.ReadFile(mfile)
			if err != nil {
				println("wrong machine.")
				os.Exit(21)
			}
			if gopsu.TrimString(string(b))[9:] != machineCode() {
				println("wrong machine.")
				os.Exit(21)
			}
		}
	}
}
