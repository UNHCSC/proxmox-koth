package lib

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/crypto/ssh"
)

func generatePair() error {
	private, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		return err
	}

	privateFile, err := os.Create(Config.SSH.PrivateKeyPath)
	if err != nil {
		return err
	}
	defer privateFile.Close()

	if err = privateFile.Chmod(0600); err != nil {
		return err
	}

	pemBlock := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(private),
	}

	if err := pem.Encode(privateFile, pemBlock); err != nil {
		return err
	}

	public, err := ssh.NewPublicKey(&private.PublicKey)
	if err != nil {
		return err
	}

	return os.WriteFile(Config.SSH.PublicKeyPath, ssh.MarshalAuthorizedKey(public), 0644)
}

var SSHPublicKey, SSHPrivateKey string

func InitSSH() error {
	// Ensure the parent directory for the key files exists
	if err := os.MkdirAll(filepath.Dir(Config.SSH.PrivateKeyPath), 0755); err != nil {
		return err
	}

	// Generate the keys if they don't exist
	if _, err := os.Stat(Config.SSH.PrivateKeyPath); os.IsNotExist(err) {
		if err := generatePair(); err != nil {
			return err
		}
	}

	// Read the keys
	pubKey, err := os.ReadFile(Config.SSH.PublicKeyPath)

	if err != nil {
		return err
	}

	privKey, err := os.ReadFile(Config.SSH.PrivateKeyPath)

	if err != nil {
		return err
	}

	SSHPublicKey = string(pubKey)
	SSHPrivateKey = string(privKey)

	return nil
}

type SSHConnection struct {
	client  *ssh.Client
	session *ssh.Session
}

func (conn *SSHConnection) Close() {
	conn.session.Close()
	conn.client.Close()
}

func (conn *SSHConnection) Send(command string) error {
	return conn.session.Run(command)
}

func (conn *SSHConnection) SendWithOutput(command string) (int, string, error) {
	output, err := conn.session.CombinedOutput(command)

	if err != nil {
		if exitErr, ok := err.(*ssh.ExitError); ok {
			return exitErr.ExitStatus(), string(output), nil
		}

		return -1, string(output), err
	}

	return 0, string(output), nil
}

func NewSSHConnection(ipAddress string) (*SSHConnection, error) {
	signer, err := ssh.ParsePrivateKey([]byte(SSHPrivateKey))
	if err != nil {
		return nil, err
	}

	config := &ssh.ClientConfig{
		User: "root",
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	client, err := ssh.Dial("tcp", ipAddress+":22", config)
	if err != nil {
		return nil, err
	}

	session, err := client.NewSession()
	if err != nil {
		client.Close()
		return nil, err
	}

	return &SSHConnection{
		client:  client,
		session: session,
	}, nil
}

func NewSSHConnectionWithRetries(ipAddress string, maxRetries int) (*SSHConnection, error) {
	var conn *SSHConnection
	var err error

	for range maxRetries {
		conn, err = NewSSHConnection(ipAddress)
		if err == nil {
			return conn, nil
		}

		time.Sleep(time.Second + time.Duration(2)*time.Second)
	}

	return nil, err
}
