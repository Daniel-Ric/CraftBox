package realms

import "strings"

type NetworkProtocol string

const (
	NetworkProtocolDefault          NetworkProtocol = "DEFAULT"
	NetworkProtocolNetherNet        NetworkProtocol = "NETHERNET"
	NetworkProtocolNetherNetJSONRPC NetworkProtocol = "NETHERNET_JSONRPC"
)

func ParseNetworkProtocol(protocol string) NetworkProtocol {
	return NetworkProtocol(strings.ToUpper(strings.TrimSpace(protocol)))
}

func (p NetworkProtocol) Valid() bool {
	switch ParseNetworkProtocol(string(p)) {
	case NetworkProtocolDefault, NetworkProtocolNetherNet, NetworkProtocolNetherNetJSONRPC:
		return true
	default:
		return false
	}
}
