//go:build !linux && !darwin && !windows

package liboc

import "os"

func getTunnelName(fd int32) (string, error) {
	return "", os.ErrInvalid
}