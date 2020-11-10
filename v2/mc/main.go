package main

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"net"
	"runtime"
	"strings"

	"github.com/xyzj/gopsu"
)

func main() {
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
	ioutil.WriteFile(".firstrun", []byte(strings.ReplaceAll(gopsu.GetRandomString(9)+base64.StdEncoding.EncodeToString(gopsu.CompressData(b.Bytes(), gopsu.ArchiveSnappy))[3:], "=", "")), 0666)
}
