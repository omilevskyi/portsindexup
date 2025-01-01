// go:build linux

package main

// linux dummy
func sysCtlUint32(string) (string, error) {
	return "", nil
}
