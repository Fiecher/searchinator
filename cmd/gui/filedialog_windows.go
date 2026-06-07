//go:build windows

package main

import (
	"runtime"
	"unsafe"

	"golang.org/x/sys/windows"
)

type openFileName struct {
	lStructSize       uint32
	hwndOwner         uintptr
	hInstance         uintptr
	lpstrFilter       *uint16
	lpstrCustomFilter *uint16
	nMaxCustFilter    uint32
	nFilterIndex      uint32
	lpstrFile         *uint16
	nMaxFile          uint32
	lpstrFileTitle    *uint16
	nMaxFileTitle     uint32
	lpstrInitialDir   *uint16
	lpstrTitle        *uint16
	flags             uint32
	nFileOffset       uint16
	nFileExtension    uint16
	lpstrDefExt       *uint16
	lCustData         uintptr
	lpfnHook          uintptr
	lpTemplateName    *uint16
	pvReserved        uintptr
	dwReserved        uint32
	flagsEx           uint32
}

const (
	ofnFileMustExist = 0x00001000
	ofnPathMustExist = 0x00000800
	ofnExplorer      = 0x00080000
	ofnNoChangeDir   = 0x00000008
)

var (
	comdlg32            = windows.NewLazySystemDLL("comdlg32.dll")
	procGetOpenFileName = comdlg32.NewProc("GetOpenFileNameW")

	shell32                 = windows.NewLazySystemDLL("shell32.dll")
	procSHBrowseForFolder   = shell32.NewProc("SHBrowseForFolderW")
	procSHGetPathFromIDList = shell32.NewProc("SHGetPathFromIDListW")

	ole32             = windows.NewLazySystemDLL("ole32.dll")
	procCoTaskMemFree = ole32.NewProc("CoTaskMemFree")
)

type browseInfo struct {
	hwndOwner      uintptr
	pidlRoot       uintptr
	pszDisplayName *uint16
	lpszTitle      *uint16
	ulFlags        uint32
	lpfn           uintptr
	lParam         uintptr
	iImage         int32
}

const (
	bifReturnOnlyFSDirs = 0x00000001
	bifEditBox          = 0x00000010
)

const nativeFileDialogAvailable = true

func nativeOpenFile(title, filterName string, exts []string) (path string, ok bool, err error) {

	pattern := ""
	for i, e := range exts {
		if i > 0 {
			pattern += ";"
		}
		pattern += "*" + e
	}
	filter := utf16Filter(filterName, pattern, "All files", "*.*")

	buf := make([]uint16, 4096)
	titlePtr, _ := windows.UTF16PtrFromString(title)

	ofn := openFileName{
		lpstrFilter: filter,
		lpstrFile:   &buf[0],
		nMaxFile:    uint32(len(buf)),
		lpstrTitle:  titlePtr,
		flags:       ofnExplorer | ofnFileMustExist | ofnPathMustExist | ofnNoChangeDir,
	}
	ofn.lStructSize = uint32(unsafe.Sizeof(ofn))

	r, _, _ := procGetOpenFileName.Call(uintptr(unsafe.Pointer(&ofn)))

	runtime.KeepAlive(filter)
	runtime.KeepAlive(buf)
	runtime.KeepAlive(titlePtr)
	runtime.KeepAlive(&ofn)

	if r == 0 {
		return "", false, nil
	}
	return windows.UTF16ToString(buf), true, nil
}

func nativeOpenFolder(title string) (path string, ok bool, err error) {
	titlePtr, _ := windows.UTF16PtrFromString(title)
	display := make([]uint16, windows.MAX_PATH)

	bi := browseInfo{
		pszDisplayName: &display[0],
		lpszTitle:      titlePtr,
		ulFlags:        bifReturnOnlyFSDirs | bifEditBox,
	}
	pidl, _, _ := procSHBrowseForFolder.Call(uintptr(unsafe.Pointer(&bi)))

	runtime.KeepAlive(titlePtr)
	runtime.KeepAlive(display)
	runtime.KeepAlive(&bi)

	if pidl == 0 {
		return "", false, nil
	}
	defer procCoTaskMemFree.Call(pidl)

	buf := make([]uint16, 4096)
	r, _, _ := procSHGetPathFromIDList.Call(pidl, uintptr(unsafe.Pointer(&buf[0])))
	runtime.KeepAlive(buf)
	if r == 0 {
		return "", false, nil
	}
	return windows.UTF16ToString(buf), true, nil
}

func utf16Filter(parts ...string) *uint16 {
	var buf []uint16
	for _, p := range parts {
		u, err := windows.UTF16FromString(p)
		if err != nil {
			continue
		}
		buf = append(buf, u...)
	}
	buf = append(buf, 0)
	return &buf[0]
}
