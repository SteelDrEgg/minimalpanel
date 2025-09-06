package http

import (
	"fmt"
	"github.com/spf13/cast"
	"github.com/zishang520/socket.io/servers/engine/v3"
	"github.com/zishang520/socket.io/servers/socket/v3"
	"github.com/zishang520/socket.io/v3/pkg/types"
	"net/http"
)

// Start 启动Socket.IO服务器
func Start(addr string) error {
	// 创建带有传输配置的Socket.IO服务器实例
	opts := socket.DefaultServerOptions()
	opts.SetPath("/socket.io")
	opts.SetTransports(types.NewSet(
		engine.Polling,
		engine.WebSocket,
	))

	server := socket.NewServer(nil, opts)
	fmt.Println("Socket.IO服务器已创建 (Engine.IO v3) - 路径: /socket.io")

	// 监听连接事件
	server.On("connection", func(clients ...any) {
		fmt.Println("Socket.io已接收")
		client := clients[0].(*socket.Socket)
		fmt.Println("客户端已连接: %s", client.Id())

		// 发送欢迎消息
		client.Emit("message", "欢迎连接到MinimalPanel Socket.IO服务器!")

		// 处理客户端消息 - 按照官方示例
		client.On("message", func(data ...any) {
			fmt.Printf("收到消息: %v", cast.ToStringSlice(data))
			// 回显消息给客户端
			client.Emit("message", data...)
		})

		// 处理断开连接
		client.On("disconnect", func(reason ...any) {
			fmt.Println("客户端 %s 断开连接，原因: %v", client.Id(), reason)
		})
	})

	http.Handle("/socket.io/", server.ServeHandler(nil))

	// 提供测试页面
	http.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("提供页面: %s %s", r.Method, r.URL.String())
		http.ServeFile(w, r, "internal/http/index.html")
	})

	fmt.Println("Socket.IO服务器启动在: http://localhost:8080")
	fmt.Println("访问测试页面: http://DrEggs-Mac-Pro-3.local:8080")

	return http.ListenAndServe(":8080", nil)
}
