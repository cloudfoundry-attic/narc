package main

import (
	"fmt"
	"os"
	"io/ioutil"
	"log"
	"github.com/cloudfoundry/sshark"
	"github.com/cloudfoundry/go_cfmessagebus"
)

func main() {
	agent := sshark.NewAgent("/tmp/warden.sock")

  pubkeyPath := fmt.Sprintf("%s/.ssh/id_rsa.pub", os.Getenv("HOME"))

  pubkey, err := ioutil.ReadFile(pubkeyPath)
  if err != nil {
    log.Fatal(err.Error())
    return
  }

  sess, err := agent.StartSession("foo")
  if err != nil {
    log.Fatal(err.Error())
    return
  }

  log.Println(sess.Port)

  err = sess.LoadPublicKey(string(pubkey))
  if err != nil {
    log.Fatal(err.Error())
    return
  }

  err = sess.StartSSHServer()
  if err != nil {
    log.Fatal(err.Error())
    return
  }
}
