package main

import (
	"strconv"
	"syscall"

	"github.com/shirou/gopsutil/net"
	"github.com/shirou/gopsutil/process"
	log "github.com/sirupsen/logrus"
)

type PortType string

const (
	UDP     PortType = "UDP"
	TCP     PortType = "TCP"
	UNKNOWN PortType = "UNKNOWN"
)

func portTypeToProtocol(t uint32) PortType {
	switch t {
	case syscall.SOCK_STREAM:
		return TCP
	case syscall.SOCK_DGRAM:
		return UDP
	}
	return UNKNOWN
}

func main() {
	conns, err := net.Connections("all")
	if err != nil {
		log.WithError(err).Error("Could not get local network listeners")
	}
	log.Info("Listing all connections", conns)

	connsByPID := make(map[int32][]*net.ConnectionStat)
	for i := range conns {
		c := conns[i]
		isIPSocket := c.Family == syscall.AF_INET || c.Family == syscall.AF_INET6
		isTCPOrUDP := c.Type == syscall.SOCK_STREAM || c.Type == syscall.SOCK_DGRAM
		// UDP doesn't have any status
		isUDPOrListening := c.Type == syscall.SOCK_DGRAM || c.Status == "LISTEN"
		// UDP is "listening" when it has a remote port of 0
		isTCPOrHasNoRemotePort := c.Type == syscall.SOCK_STREAM || c.Raddr.Port == 0
		// PID of 0 means that the listening file descriptor couldn't be mapped
		// back to a process's set of open file descriptors in /proc
		if !isIPSocket || !isTCPOrUDP || !isUDPOrListening || !isTCPOrHasNoRemotePort || c.Pid == 0 {
			log.Info("Skipping conn: ", c)
			continue
		}
		connsByPID[c.Pid] = append(connsByPID[c.Pid], &c)
	}

	for pid, conns := range connsByPID {
		proc, err := process.NewProcess(pid)

		if err != nil {
			log.WithFields(log.Fields{
				"pid": pid,
				"err": err,
			}).Warn("Could not examine process (it might have terminated already)")
			continue
		}

		name, err := proc.Name()
		if err != nil {
			log.WithField("pid", pid).Error("Could not get process name")
			continue
		}

		args, err := proc.Cmdline()
		if err != nil {
			log.WithField("pid", pid).Error("Could not get process args")
			continue
		}

		dims := map[string]string{
			"pid": strconv.Itoa(int(pid)),
		}

		for _, c := range conns {
			log.Infof("Endpoint: %s-%d-%s-%d Name: %s Dims: %v", c.Laddr.IP, c.Laddr.Port, portTypeToProtocol(c.Type), pid, name, dims)
			log.Info("Endpoint command", args)

			ip := c.Laddr.IP
			// An IP addr of 0.0.0.0 means it listens on all interfaces, including
			// localhost, so use that since we can't actually connect to 0.0.0.0.
			if ip == "0.0.0.0" {
				ip = "127.0.0.1"
			}
			log.Infof("Endpoint Net Info: %s, %d, %s", ip, c.Laddr.Port, portTypeToProtocol(c.Type))
		}
	}
}
