//go:build !windows

package utils

func IsHiddenFile(filename string) (bool, error) {
	return filename[0:1] == ".", nil
}
