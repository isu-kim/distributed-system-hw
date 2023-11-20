package relay

import (
	"log"
	"net"
)

func TempRelay(conn net.Conn) {
	log.Printf("Relay from %s", conn.RemoteAddr())
}
