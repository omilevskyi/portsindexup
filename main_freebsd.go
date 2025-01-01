// go:build freebsd

package main

import (
	"encoding/binary"
	"fmt"

	"golang.org/x/sys/unix"
)

func sysCtlUint32(param string) (string, error) {
	data, err := unix.SysctlRaw(param) // unix.Sysctl(param) does not work for osreldate
	if err != nil {
		return "", fmt.Errorf("error reading sysctl: %w", err)
	}

	if len(data) < 4 {
		return "", fmt.Errorf("unexpected data length: %d", len(data))
	}

	return fmt.Sprint(binary.LittleEndian.Uint32(data)), nil // FreeBSD amd64 has Little Endian
}
