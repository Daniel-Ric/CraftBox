package desktop

import (
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
)

func minecraftJoinURI(address string) (string, error) {
	if strings.Contains(address, "://") {
		u, e := url.Parse(address)
		if e != nil || u.User != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Path != "" || u.RawQuery != "" || u.Fragment != "" {
			return "", fmt.Errorf("invalid local Minecraft endpoint")
		}
		address = u.Host
	}
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return "", err
	}
	n, err := strconv.Atoi(port)
	if err != nil || n < 1 || n > 65535 {
		return "", fmt.Errorf("Minecraft connection port must be between 1 and 65535")
	}
	if host == "" || host == "0.0.0.0" || host == "::" {
		host = "127.0.0.1"
	}
	if strings.ContainsAny(host, " \t\r\n/?#@\\") {
		return "", fmt.Errorf("invalid Minecraft connection hostname")
	}
	return "minecraft://connect?" + url.Values{"serverUrl": {host}, "serverPort": {port}}.Encode(), nil
}

func minecraftJoinForState(state State) (string, error) {
	if !state.Running || state.Stopping || !state.MinecraftJoinReady {
		return "", fmt.Errorf("Start a capture and wait until its Minecraft connection is ready")
	}
	return minecraftJoinURI(state.ListenAddr)
}
