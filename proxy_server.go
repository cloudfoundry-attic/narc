package narc

import (
	"code.google.com/p/go.net/websocket"
	"fmt"
	"github.com/kr/pty"
	"io"
	"net"
	"net/http"
	"os/exec"
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
	taskID := ws.Request().Header.Get("X-Task-ID")

	task, found := p.agent.Registry.Lookup(taskID)
	if !found {
		// TODO: don't panic
		panic("session not found")
	}

	wshdSocket := fmt.Sprintf("/opt/warden/containers/%s/run/wshd.sock", task.Container.ID())

	c := exec.Command(
		"sudo",
		"/opt/warden/warden/root/linux/skeleton/bin/wsh",
		"--socket", wshdSocket,
		"--user", "vcap",
	)

	pty, err := pty.Start(c)
	if err != nil {
		// TODO: don't panic
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
		err = p.agent.StopTask(taskID)
		if err != nil {
			panic(err)
		}

		ws.Close()
		<-connectionClosed

	case <-connectionClosed:
	}
}
