package deploy

import (
	"code.google.com/p/go.crypto/ssh"
	"log"
)

func newSSHClient(host string, sshConfig *ssh.ClientConfig) (*ssh.Client, error) {
	client, err := ssh.Dial("tcp", host, sshConfig)
	if err != nil {
		log.Println("ssh.Dial failed", err)
		return nil, err
	}

	return client, nil
}

func newSSHClientConfig(user string, key []byte) (*ssh.ClientConfig, error) {
	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		return nil, err
	}

	config := &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
	}
	return config, nil
}
