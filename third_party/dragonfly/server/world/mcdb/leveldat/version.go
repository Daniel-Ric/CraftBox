package leveldat

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"strconv"
	"strings"
)

const Version = 10

var minimumCompatibleClientVersion []int32

func init() {
	fullVersion := append(strings.Split(protocol.CurrentVersion, "."), "0", "0")
	for _, v := range fullVersion {
		i, _ := strconv.Atoi(v)
		minimumCompatibleClientVersion = append(minimumCompatibleClientVersion, int32(i))
	}
}
