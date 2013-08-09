package narc

import (
	"code.google.com/p/go.net/websocket"
	"fmt"
	"io"
	"net"
	"net/http"
)

type ProxyServer struct {
	agent *Agent

	listener net.Listener
}

func NewProxyServer(agent *Agent) *ProxyServer {
	return &ProxyServer{
		agent: agent,
	}
}

func (p *ProxyServer) Start(port int) error {
	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: websocket.Handler(p.handler),
	}

	listener, err := net.Listen("tcp", server.Addr)
	if err != nil {
		return err
	}

	p.listener = listener

	go server.Serve(listener)

	return nil
}

func (p *ProxyServer) Stop() error {
	if p.listener != nil {
		return p.listener.Close()
	}

	return nil
}

func (p *ProxyServer) handler(ws *websocket.Conn) {
	defer ws.Close()

	taskID := ws.Request().Header.Get("X-Task-ID")
	secureToken := ws.Request().Header.Get("X-Task-Token")

	task, found := p.agent.Registry.Lookup(taskID)

	if !found {
		ws.Write([]byte("Unknown Task\n"))
		return
	}

	if task.SecureToken != secureToken {
		ws.Write([]byte("Invalid Token\n"))
		return
	}

	pty, err := task.Run()
	if err != nil {
		panic(err)
	}

	commandDone := make(chan bool)
	connectionClosed := make(chan bool)

	go func() {
		io.Copy(ws, pty)
		commandDone <- true
	}()

	go func() {
		io.Copy(pty, ws)
		connectionClosed <- true
	}()

	select {
	case <-commandDone:
		err := p.agent.StopTask(taskID)
		if err != nil {
			panic(err)
		}

		<-connectionClosed

	case <-connectionClosed:
	}
}
