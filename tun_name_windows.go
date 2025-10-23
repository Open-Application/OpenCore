//go:build windows

package liboc

import (
	"fmt"
	"golang.org/x/sys/windows"
)

var (
	modiphlpapi          = windows.NewLazySystemDLL("iphlpapi.dll")
	procGetAdaptersAddrs = modiphlpapi.NewProc("GetAdaptersAddresses")
)

func getTunnelName(fd int32) (string, error) {
	return fmt.Sprintf("tun%d", fd), nil
}