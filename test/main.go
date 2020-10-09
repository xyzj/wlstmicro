package main

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"runtime"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/render"
	"github.com/xyzj/gopsu"
)

var (
	templateHelloWorld = `<!DOCTYPE html>
<!-- saved from url=(0026)http://css-tricks.com/0404 -->
<html>
<head>
<meta http-equiv="Content-Type" content="text/html;charset=UTF-8">
  <title>E komo mai</title>
  <style>
        body {margin: 0; padding: 20px; text-align:center; font-family:Arial, Helvetica, sans-serif; font-size:26px; background-color:#1e1e1e;}
        </style>
</head>
<body>
    <img  src="http://10.3.7.16:6819/m/abcdefg">
</body></html>`
)

func test1(c *gin.Context) {
	println(c.Param("spath"))
	b, _ := ioutil.ReadFile("/home/xy/Pictures/wallpaper/batman.png")
	c.Writer.Write(b)
}

func test2(c *gin.Context) {
	c.Header("Content-Type", "text/html")
	c.Status(http.StatusOK)
	render.WriteString(c.Writer, templateHelloWorld, nil)
}
func aaa(args ...interface{}) {
	for k, v := range args {
		println(fmt.Sprintf("%d, %+v", k, v))
	}
	println("Done")
}

// func engine() *gin.Engine {
// 	r := wlstmicro.NewHTTPEngine(ginmiddleware.ReadParams())
// 	r.GET("/aaa", test2)
// 	r.GET("/m/:spath", test1)
// 	return r
// }

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
