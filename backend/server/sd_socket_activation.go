package server

import (
	"net"
	"sync"

	"github.com/coreos/go-systemd/v22/activation"
)

var (
	systemdSockets     map[string][]net.Listener
	systemdSocketsErr  error
	systemdSocketsOnce sync.Once
)

func initSystemdSockets() {
	fds, err := activation.ListenersWithNames()

	if err != nil {
		systemdSockets = nil
		systemdSocketsErr = err
	} else {
		systemdSockets = fds
		systemdSocketsErr = nil
	}
}

func getSystemdSockets() (map[string][]net.Listener, error) {
	systemdSocketsOnce.Do(initSystemdSockets)
	if systemdSocketsErr != nil {
		return nil, systemdSocketsErr
	}

	return systemdSockets, nil
}
