//go:build !windows

package liboc

import (
	"net"
	"syscall"

	"golang.org/x/sys/unix"
)

func dup(fd int) (nfd int, err error) {
	return syscall.Dup(fd)
}

func linkFlags(flags uint32) net.Flags {
	var netFlags net.Flags
	if flags&unix.IFF_UP != 0 {
		netFlags |= net.FlagUp
	}
	if flags&unix.IFF_BROADCAST != 0 {
		netFlags |= net.FlagBroadcast
	}
	if flags&unix.IFF_LOOPBACK != 0 {
		netFlags |= net.FlagLoopback
	}
	if flags&unix.IFF_POINTOPOINT != 0 {
		netFlags |= net.FlagPointToPoint
	}
	if flags&unix.IFF_MULTICAST != 0 {
		netFlags |= net.FlagMulticast
	}
	return netFlags
}