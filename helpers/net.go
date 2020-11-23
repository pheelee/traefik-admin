package helpers

import (
	"errors"
	"net"
)

//GetHostIP returns the ip address of the first non loopback interface
func GetHostIP() (string, error) {
	var (
		addrs []net.Addr
		err   error
	)
	if addrs, err = net.InterfaceAddrs(); err != nil {
		return "", err
	}

	for _, a := range addrs {
		if ipnet, ok := a.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String(), nil
			}
		}
	}
	return "", errors.New("No ip found")
}
