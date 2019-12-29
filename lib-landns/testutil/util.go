package testutil

import (
	"fmt"
	"math/rand"
	"net"
)

const (
	PortMin = 49152
	PortMax = 65535
)

// FindEmptyPort is find unused TCP port.
func FindEmptyPort() int {
	for {
		port := rand.Intn(PortMax-PortMin+1) + PortMin
		l, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", port))
		if err == nil {
			l.Close()
			return port
		}
	}
}
