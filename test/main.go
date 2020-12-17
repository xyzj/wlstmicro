package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"sync/atomic"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/render"
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
	var id uint64
	for i := 0; i < 10; i++ {
		println(atomic.AddUint64(&id, 1))
	}
}
