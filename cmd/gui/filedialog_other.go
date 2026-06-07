//go:build !windows

package main

const nativeFileDialogAvailable = false

func nativeOpenFile(title, filterName string, exts []string) (string, bool, error) {
	return "", false, nil
}

func nativeOpenFolder(title string) (string, bool, error) {
	return "", false, nil
}
