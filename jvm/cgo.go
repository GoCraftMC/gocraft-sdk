package jvm

// #include "bridge.h"
import "C"

import (
	"sync/atomic"
	"unsafe"
)

var (
	booted atomic.Bool
	token  int64
	modID  string
)

func requireBooted() {
	if !booted.Load() {
		panic("gocraft: the JVM was used before the mod was booted; move this call from init() into an entrypoint such as gocraft.OnInitialize")
	}
}

func setTable(table int64) { C.c_set_table(C.i64(table)) }

func addr(b []byte) unsafe.Pointer {
	if len(b) == 0 {
		return nil
	}
	return unsafe.Pointer(&b[0])
}

func strAddr(s string) unsafe.Pointer {
	if len(s) == 0 {
		return nil
	}
	return unsafe.Pointer(unsafe.StringData(s))
}

func ptrAddr[T any](p *T) unsafe.Pointer { return unsafe.Pointer(p) }
