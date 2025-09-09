package netx

import (
	"fmt"
	"github.com/zishang520/socket.io/servers/engine/v3"
	"github.com/zishang520/socket.io/servers/socket/v3"
	"github.com/zishang520/socket.io/v3/pkg/types"
	"net/http"
)

// Socket represents a wrapper around the Socket.IO server
type Socket struct {
	sock       *socket.Server
	Namespaces map[string]Namespace
}

// Initialize configures and creates the Socket.IO server
func (self *Socket) Initialize() {
	opts := socket.DefaultServerOptions()
	opts.SetPath("/socket.io")
	opts.SetTransports(types.NewSet(
		engine.Polling,   // HTTP long-polling transport
		engine.WebSocket, // WebSocket transport for real-time communication
	))
	opts.SetMaxHttpBufferSize(1e7) // 10MB
	self.sock = socket.NewServer(nil, opts)
	self.Namespaces = make(map[string]Namespace)
}

// AddNamespace creates a new Socket.IO namespace and adds it to the server
func (self *Socket) AddNamespace(name string) {
	namespace := Namespace{namespace: self.sock.Of(name, nil)}
	namespace.Initialize()
	self.Namespaces[name] = namespace
}

// GetNamespace returns the desired namespace
func (self *Socket) GetNamespace(name string) Namespace {
	return self.Namespaces[name]
}

// Handler returns an HTTP handler for the Socket.IO server
func (self *Socket) Handler() http.Handler {
	return self.sock.ServeHandler(nil)
}

// Namespace represents a Socket.IO namespace with custom event handling
type Namespace struct {
	namespace socket.Namespace
	events    map[string]func(client *socket.Socket, data ...any)
	//middileWare []func(client *socket.Socket, next func())
}

// Initialize sets up the namespace with default event handlers
func (self *Namespace) Initialize() {
	self.events = map[string]func(*socket.Socket, ...any){
		"disconnect": func(client *socket.Socket, reason ...any) {},
	}
}

// AddEvent registers a custom event handler for the namespace
func (self *Namespace) AddEvent(event string, f func(*socket.Socket, ...any)) {
	self.events[event] = f
}

// RegisterEvents activates all the event handlers for new client connections
func (self *Namespace) RegisterEvents() {
	self.namespace.On("connection", func(clients ...any) {
		client := clients[0].(*socket.Socket)
		for event, f := range self.events {
			client.On(event, func(data ...any) { f(client, data) })
		}
	})
}

// AddMiddleware adds a middleware to the namespace
func (self *Namespace) AddMiddleware(f func(client *socket.Socket, next func(*socket.ExtendedError))) {
	self.namespace.Use(f)
}

// Test function, ignore this
func Start(addr string) error {
	server := new(Socket)
	server.Initialize()
	server.AddNamespace("/ttt")

	defaultNamespace := server.Namespaces["/ttt"]

	defaultNamespace.AddEvent("message", func(client *socket.Socket, data ...any) {
		client.Emit("message", data...)
	})
	defaultNamespace.RegisterEvents()

	defaultNamespace.AddMiddleware(func(client *socket.Socket, next func(*socket.ExtendedError)) {
		fmt.Println(client.Handshake().Auth)
		next(nil)
	})
	http.Handle("/socket.io/", server.Handler())

	fmt.Println("Socket.IO服务器启动在: http://localhost:8080")
	fmt.Println("访问测试页面: http://DrEggs-Mac-Pro-3.local:8080")
	StartFrontend()

	return http.ListenAndServe(":8080", nil)
}
