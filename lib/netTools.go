package lib

import (
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"runtime"
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
