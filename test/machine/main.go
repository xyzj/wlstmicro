package main

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"net"
	"runtime"
	"strings"

	"github.com/xyzj/gopsu"
)

func main() {
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
	println(b.String())
	println(gopsu.CodeString(base64.StdEncoding.EncodeToString(gopsu.DoZlibCompress(b.Bytes()))))
}
