package narc

import (
	"code.google.com/p/go.crypto/ssh"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io"
	"log"
)

type ProxyServer struct {
	agent   *Agent
	hostKey []byte

	listener *ssh.Listener
}

func NewProxyServer(agent *Agent) (*ProxyServer, error) {
	key, err := generateHostKey()
	if err != nil {
		return nil, err
	}

	return &ProxyServer{
		agent:   agent,
		hostKey: key,
	}, nil
}

func (p *ProxyServer) Start(port int) error {
	config := &ssh.ServerConfig{
		PasswordCallback: p.verifyTaskAccess,
	}

	err := config.SetRSAPrivateKey(p.hostKey)
	if err != nil {
		return err
	}

	l, err := ssh.Listen("tcp", fmt.Sprintf(":%d", port), config)
	if err != nil {
		return err
	}

	p.listener = l

	log.Println("listening on port", port)

	go p.serveConnections()

	return nil
}

func (p *ProxyServer) Stop() error {
	if p.listener != nil {
		return p.listener.Close()
	}

	return nil
}

func (p *ProxyServer) verifyTaskAccess(conn *ssh.ServerConn, user, password string) bool {
	log.Println("verifying:", user, password)

	task, found := p.agent.Registry.Lookup(user)
	if !found {
		log.Println("verify failed: task not found")
		return false
	}

	return task.SecureToken == password
}

func (p *ProxyServer) serveConnections() {
	for {
		conn, err := p.listener.Accept()
		if err != nil {
			log.Println("error accepting connection:", err)
			break
		}

		log.Println("accepted connection")

		err = conn.Handshake()
		if err != nil {
			log.Println("handshake failed:", err)
			continue
		}

		go p.handleSession(conn)
	}
}

func (p *ProxyServer) handleSession(conn *ssh.ServerConn) {
	defer conn.Close()

	for {
		channel, err := conn.Accept()
		if err == io.EOF {
			return
		}

		if err != nil {
			log.Println("failed to accept ssh channel", err)
			return
		}

		if channel.ChannelType() != "session" {
			channel.Reject(ssh.UnknownChannelType, "unknown channel type")
			break
		}

		go p.handleChannel(channel, conn.User)
	}
}

func (p *ProxyServer) handleChannel(channel ssh.Channel, taskID string) {
	err := channel.Accept()
	if err != nil {
		log.Println("failed to accept channel request:", err)
		return
	}

	defer channel.Close()

	task, found := p.agent.Registry.Lookup(taskID)
	if !found {
		log.Println("unknown task:", task)
		return
	}

	commandDone, channelClosed, err := task.Attach(channel)
	if err != nil {
		log.Println("failed to execute task:", err)
		return
	}

	select {
	case <-commandDone:
		err := p.agent.StopTask(taskID)
		if err != nil {
			panic(err)
		}

		<-channelClosed

	case <-channelClosed:
	}
}

func generateHostKey() ([]byte, error) {
	key, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		return nil, err
	}

	blk := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	}

	return pem.EncodeToMemory(blk), nil
}
