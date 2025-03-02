package lib

import (
	"crypto/rand"
	"fmt"
	"io"
	"net"
	"net/http"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

func PingHost(ipAddress string) bool {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("ping", "-n", "1", "-w", "2000", ipAddress)
	default:
		cmd = exec.Command("ping", "-c", "1", "-W", "2", ipAddress)
	}

	if err := cmd.Run(); err != nil {
		return false
	}

	return true
}

func WaitOnline(ipAddress string) error {
	start := time.Now()

	for time.Since(start) < 3*time.Minute {
		if PingHost(ipAddress) {
			return nil
		}

		time.Sleep(time.Second * 3)
	}

	return fmt.Errorf("host not available on %s within timeout", ipAddress)
}

func HttpGetHost(fullURL string) string {
	resp, err := http.Get(fullURL)

	if err != nil {
		return ""
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)

	if err != nil {
		return ""
	}

	return string(body)
}

func GetLocalIP() (string, error) {
	output, err := exec.Command("hostname", "-I").Output()

	if err != nil {
		interfaces, err := net.Interfaces()

		if err != nil {
			return "", err
		}

		for _, iface := range interfaces {
			addrs, err := iface.Addrs()
			if err != nil {
				continue
			}

			for _, addr := range addrs {
				if ipNet, ok := addr.(*net.IPNet); ok && !ipNet.IP.IsLoopback() && ipNet.IP.To4() != nil {
					return strings.TrimSpace(ipNet.IP.String()), nil
				}
			}
		}

		return "", fmt.Errorf("no valid IP address found")
	}

	return strings.TrimSpace(string(output)), nil
}

func RandomString(length int) string {
	bytes := make([]byte, length)

	_, err := rand.Read(bytes)

	if err != nil {
		return ""
	}

	return fmt.Sprintf("%x", bytes)
}
