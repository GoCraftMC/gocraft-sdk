package jvm

// #include <stdint.h>
import "C"

import (
	"fmt"
	"runtime/debug"
	"sync"
	"unsafe"
)

const abiVersion = 1

const (
	statusOK       = 0
	statusThrown   = 1
	statusPanicked = 2
)

//export gocraft_abi
func gocraft_abi() C.int64_t { return abiVersion }

var (
	entryMu  sync.Mutex
	entries  = map[string][]func(){}
	bootOnce []func()
)

// OnEntry registers fn for a Fabric entrypoint key such as "main" or "client". Handlers run in
// registration order; it is safe to call from package init.
func OnEntry(key string, fn func()) {
	entryMu.Lock()
	defer entryMu.Unlock()
	entries[key] = append(entries[key], fn)
}

// OnBoot runs fn once the bridge is usable, or right away if it already is.
func OnBoot(fn func()) {
	entryMu.Lock()
	if !booted.Load() {
		bootOnce = append(bootOnce, fn)
		entryMu.Unlock()
		return
	}
	entryMu.Unlock()
	fn()
}

//export gocraft_boot
func gocraft_boot(table, tok C.int64_t, name unsafe.Pointer, nameLen C.int64_t) {
	setTable(int64(table))
	token = int64(tok)
	modID = string(unsafe.Slice((*byte)(name), int(nameLen)))
	startReleaser()

	entryMu.Lock()
	booted.Store(true)
	pending := bootOnce
	bootOnce = nil
	entryMu.Unlock()

	for _, fn := range pending {
		fn()
	}
}

//export gocraft_entry
func gocraft_entry(name unsafe.Pointer, nameLen C.int64_t, ret unsafe.Pointer) (status C.int) {
	defer func() {
		if r := recover(); r != nil {
			status = encodePanic(r, ret)
		}
	}()

	key := string(unsafe.Slice((*byte)(name), int(nameLen)))
	entryMu.Lock()
	handlers := entries[key]
	entryMu.Unlock()

	for _, fn := range handlers {
		fn()
	}
	return statusOK
}

//export gocraft_dispatch
func gocraft_dispatch(cb C.int64_t, args unsafe.Pointer, n C.int64_t, ret unsafe.Pointer) (status C.int) {
	defer func() {
		if r := recover(); r != nil {
			status = encodePanic(r, ret)
		}
	}()

	entry := lookupCallback(int64(cb))
	frame := Args{kinds: entry.params, slots: unsafe.Slice((*int64)(args), int(n))}
	result := entry.fn(&frame)
	*(*int64)(ret) = returnSlot(entry.ret, result)
	return statusOK
}

// encodePanic turns a recovered value into a status for Java. A Java exception that unwound through
// Go goes back unchanged, so Java sees its original type and stack.
func encodePanic(r any, ret unsafe.Pointer) C.int {
	out := (*int64)(ret)

	if thrown, ok := r.(*Throwable); ok {
		*out = dupHandle(thrown.Object)
		return statusThrown
	}

	*out = newStringHandle(fmt.Sprintf("%v\n\n%s", r, debug.Stack()))
	return statusPanicked
}
