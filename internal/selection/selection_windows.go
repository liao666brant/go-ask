package selection

import (
	"context"
	"runtime"
	"strings"
	"syscall"
	"unicode/utf16"
	"unsafe"
)

var (
	user32             = syscall.NewLazyDLL("user32.dll")
	getGUIThreadInfo   = user32.NewProc("GetGUIThreadInfo")
	getClassName       = user32.NewProc("GetClassNameW")
	getWindowLong      = user32.NewProc("GetWindowLongW")
	sendMessageTimeout = user32.NewProc("SendMessageTimeoutW")
	ole32              = syscall.NewLazyDLL("ole32.dll")
	oleAut32           = syscall.NewLazyDLL("oleaut32.dll")
	coInitializeEx     = ole32.NewProc("CoInitializeEx")
	coUninitialize     = ole32.NewProc("CoUninitialize")
	coCreateInstance   = ole32.NewProc("CoCreateInstance")
	sysStringLen       = oleAut32.NewProc("SysStringLen")
	sysFreeString      = oleAut32.NewProc("SysFreeString")

	clsidCUIAutomation = syscall.GUID{Data1: 0xff48dba4, Data2: 0x60ef, Data3: 0x4201, Data4: [8]byte{0xaa, 0x87, 0x54, 0x10, 0x3e, 0xef, 0x59, 0x4e}}
	iidUIAutomation    = syscall.GUID{Data1: 0x30cbe57d, Data2: 0xd9d0, Data3: 0x452a, Data4: [8]byte{0xab, 0x13, 0x7a, 0xc5, 0xac, 0x48, 0x25, 0xee}}
	iidTextPattern     = syscall.GUID{Data1: 0x32eba289, Data2: 0x3583, Data3: 0x42c9, Data4: [8]byte{0x9c, 0x59, 0x3b, 0x6d, 0x9a, 0x1e, 0x9b, 0x6a}}
)

// Slots from the IUIAutomation* interfaces in UIAutomationClient.h.
const (
	releaseObject          = 2
	getFocusedElement      = 8
	getCurrentIsPassword   = 35
	getCurrentNativeWindow = 36
	getCurrentPatternAs    = 14
	getSelection           = 5
	getLength              = 3
	getRange               = 4
	getText                = 12
	textPatternID          = 10014
)

type automationObject struct{ methods *[37]uintptr }

func (o *automationObject) release() {
	syscall.SyscallN(o.methods[releaseObject], uintptr(unsafe.Pointer(o)))
}

// Use the focused control's UI Automation TextPattern to read its selection.
// This never synthesizes a copy keystroke or touches the user's clipboard.
func Text(ctx context.Context) string {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	hr, _, _ := coInitializeEx.Call(0, 0) // COINIT_MULTITHREADED
	if int32(hr) < 0 {
		return ""
	}
	defer coUninitialize.Call()

	var automation, element, pattern, ranges *automationObject
	hr, _, _ = coCreateInstance.Call(uintptr(unsafe.Pointer(&clsidCUIAutomation)), 0, 1, uintptr(unsafe.Pointer(&iidUIAutomation)), uintptr(unsafe.Pointer(&automation)))
	if int32(hr) < 0 || automation == nil {
		return ""
	}
	defer automation.release()
	hr, _, _ = syscall.SyscallN(automation.methods[getFocusedElement], uintptr(unsafe.Pointer(automation)), uintptr(unsafe.Pointer(&element)))
	if int32(hr) < 0 || element == nil {
		return ""
	}
	defer element.release()
	var password int32
	hr, _, _ = syscall.SyscallN(element.methods[getCurrentIsPassword], uintptr(unsafe.Pointer(element)), uintptr(unsafe.Pointer(&password)))
	if int32(hr) >= 0 && password != 0 {
		return ""
	}
	hr, _, _ = syscall.SyscallN(element.methods[getCurrentPatternAs], uintptr(unsafe.Pointer(element)), textPatternID, uintptr(unsafe.Pointer(&iidTextPattern)), uintptr(unsafe.Pointer(&pattern)))
	if int32(hr) < 0 || pattern == nil {
		return selectedClassicEdit(ctx, element)
	}
	defer pattern.release()
	hr, _, _ = syscall.SyscallN(pattern.methods[getSelection], uintptr(unsafe.Pointer(pattern)), uintptr(unsafe.Pointer(&ranges)))
	if int32(hr) < 0 || ranges == nil {
		return ""
	}
	defer ranges.release()
	var count int32
	hr, _, _ = syscall.SyscallN(ranges.methods[getLength], uintptr(unsafe.Pointer(ranges)), uintptr(unsafe.Pointer(&count)))
	if int32(hr) < 0 {
		return ""
	}
	var parts []string
	for i := int32(0); i < count && ctx.Err() == nil; i++ {
		var r *automationObject
		hr, _, _ = syscall.SyscallN(ranges.methods[getRange], uintptr(unsafe.Pointer(ranges)), uintptr(i), uintptr(unsafe.Pointer(&r)))
		if int32(hr) < 0 || r == nil {
			continue
		}
		var bstr *uint16
		hr, _, _ = syscall.SyscallN(r.methods[getText], uintptr(unsafe.Pointer(r)), ^uintptr(0), uintptr(unsafe.Pointer(&bstr)))
		r.release()
		if int32(hr) >= 0 && bstr != nil {
			length, _, _ := sysStringLen.Call(uintptr(unsafe.Pointer(bstr)))
			parts = append(parts, string(utf16.Decode(unsafe.Slice(bstr, int(length)))))
			sysFreeString.Call(uintptr(unsafe.Pointer(bstr)))
		}
	}
	return strings.TrimSpace(strings.Join(parts, "\n"))
}

// Standard Win32 EDIT controls can expose a UIA element without TextPattern.
// EM_GETSEL and WM_GETTEXT are system messages that Windows marshals between
// processes. RichEdit and custom classes have different contracts.
func selectedClassicEdit(ctx context.Context, element *automationObject) string {
	var hwnd uintptr
	hr, _, _ := syscall.SyscallN(element.methods[getCurrentNativeWindow], uintptr(unsafe.Pointer(element)), uintptr(unsafe.Pointer(&hwnd)))
	if int32(hr) < 0 || hwnd == 0 || hwnd != keyboardFocus() || ctx.Err() != nil {
		return ""
	}
	var class [32]uint16
	getClassName.Call(hwnd, uintptr(unsafe.Pointer(&class[0])), uintptr(len(class)))
	if !strings.EqualFold(syscall.UTF16ToString(class[:]), "Edit") {
		return ""
	}
	const esPassword = 0x20
	style, _, _ := getWindowLong.Call(hwnd, ^uintptr(15)) // GWL_STYLE (-16)
	if style == 0 || style&esPassword != 0 {
		return ""
	}
	const (
		emGetPasswordChar = 0xd2
		emGetSel          = 0xb0
		wmGetText         = 0x0d
		abortIfHung       = 2
	)
	var result uintptr
	ok, _, _ := sendMessageTimeout.Call(hwnd, emGetPasswordChar, 0, 0, abortIfHung, 120, uintptr(unsafe.Pointer(&result)))
	if ok == 0 || result != 0 {
		return ""
	}
	var start, end uint32
	ok, _, _ = sendMessageTimeout.Call(hwnd, emGetSel, uintptr(unsafe.Pointer(&start)), uintptr(unsafe.Pointer(&end)), abortIfHung, 120, uintptr(unsafe.Pointer(&result)))
	if ok == 0 || start >= end || ctx.Err() != nil {
		return ""
	}
	// WM_GETTEXT reads from the beginning. Include only the prefix through
	// the end of the selection, plus its required terminating UTF-16 NUL.
	buf := make([]uint16, int(end)+1)
	ok, _, _ = sendMessageTimeout.Call(hwnd, wmGetText, uintptr(len(buf)), uintptr(unsafe.Pointer(&buf[0])), abortIfHung, 120, uintptr(unsafe.Pointer(&result)))
	if ok == 0 || result < uintptr(end) || hwnd != keyboardFocus() {
		return ""
	}
	return strings.TrimSpace(string(utf16.Decode(buf[start:end])))
}

func keyboardFocus() uintptr {
	var info struct {
		size, flags                               uint32
		active, focus, capture, menu, move, caret uintptr
		rect                                      [4]int32
	}
	info.size = uint32(unsafe.Sizeof(info))
	ok, _, _ := getGUIThreadInfo.Call(0, uintptr(unsafe.Pointer(&info)))
	if ok == 0 {
		return 0
	}
	return info.focus
}
