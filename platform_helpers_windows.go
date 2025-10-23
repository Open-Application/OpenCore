//go:build windows

package liboc

import (
	"net"
	"syscall"

	"golang.org/x/sys/windows"
)

func dup(fd int) (nfd int, err error) {
	currentProcess, err := windows.GetCurrentProcess()
	if err != nil {
		return 0, err
	}

	var duplicatedHandle windows.Handle
	err = windows.DuplicateHandle(
		currentProcess,
		windows.Handle(fd),
		currentProcess,
		&duplicatedHandle,
		0,
		false,
		windows.DUPLICATE_SAME_ACCESS,
	)
	if err != nil {
		return 0, err
	}

	return int(duplicatedHandle), nil
}

func linkFlags(flags uint32) net.Flags {
	var netFlags net.Flags

	const (
		IF_TYPE_SOFTWARE_LOOPBACK = 24
		IF_OPER_STATUS_UP         = 1
	)

	if flags&syscall.IFF_UP != 0 {
		netFlags |= net.FlagUp
	}
	if flags&syscall.IFF_BROADCAST != 0 {
		netFlags |= net.FlagBroadcast
	}
	if flags&syscall.IFF_LOOPBACK != 0 {
		netFlags |= net.FlagLoopback
	}
	if flags&syscall.IFF_POINTTOPOINT != 0 {
		netFlags |= net.FlagPointToPoint
	}
	if flags&syscall.IFF_MULTICAST != 0 {
		netFlags |= net.FlagMulticast
	}

	return netFlags
}