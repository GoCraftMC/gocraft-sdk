package jvm

// #include "bridge.h"
import "C"

import (
	"fmt"
	"math"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"unsafe"
)

// Callback is a Go function callable from Java. Its return value is converted to the Java method's
// return type; for void methods it is ignored.
type Callback func(a *Args) any

// Impl maps "name(descriptor)ret" to the Go function implementing it.
type Impl map[string]Callback

type callbackEntry struct {
	fn     Callback
	params string
	ret    byte
}

var (
	callbacks      sync.Map
	nextCallbackID atomic.Int64
	liveCallbacks  atomic.Int64
)

func registerCallback(fn Callback, params string, ret byte) int64 {
	id := nextCallbackID.Add(1)
	callbacks.Store(id, &callbackEntry{fn: fn, params: params, ret: ret})
	liveCallbacks.Add(1)
	return id
}

func forgetCallback(id int64) {
	if _, loaded := callbacks.LoadAndDelete(id); loaded {
		liveCallbacks.Add(-1)
	}
}

// CallbackCount returns how many Go functions Java can currently call, for finding leaks. A proxy's
// functions are freed once both Go and Java have dropped it. A closure holding a reference to its own
// proxy keeps it alive, because the two runtimes' collectors cannot see a cycle spanning both.
func CallbackCount() int64 { return liveCallbacks.Load() }

func lookupCallback(id int64) *callbackEntry {
	entry, ok := callbacks.Load(id)
	if !ok {
		panic(fmt.Sprintf("jvm: unknown callback %d", id))
	}
	return entry.(*callbackEntry)
}

// Args holds the arguments of a call from Java. It is only valid during the callback.
type Args struct {
	kinds string
	slots []int64
	objs  []Object
}

// Len returns the number of arguments.
func (a *Args) Len() int { return len(a.slots) }

// Object returns argument i as a reference that may outlive the callback. Taking it zeroes the slot,
// which tells Java that Go now owns the handle.
func (a *Args) Object(i int) Object {
	if a.objs == nil {
		a.objs = make([]Object, len(a.slots))
	}
	if a.slots[i] != 0 {
		a.objs[i] = wrap(a.slots[i])
		a.slots[i] = 0
	}
	return a.objs[i]
}

// String returns argument i as a Go string, or its toString() for other objects.
func (a *Args) String(i int) string { return a.Object(i).String() }

// Int returns argument i as an int32.
func (a *Args) Int(i int) int32 { return int32(a.slots[i]) }

// Long returns argument i as an int64.
func (a *Args) Long(i int) int64 { return a.slots[i] }

// Bool returns argument i as a bool.
func (a *Args) Bool(i int) bool { return a.slots[i] != 0 }

// Float returns argument i as a float32.
func (a *Args) Float(i int) float32 { return math.Float32frombits(uint32(a.slots[i])) }

// Double returns argument i as a float64.
func (a *Args) Double(i int) float64 { return math.Float64frombits(uint64(a.slots[i])) }

// returnSlot converts a callback's result. References are handed over with a fresh handle because
// the Go wrapper may be collected before Java reads it; Java releases that handle.
func returnSlot(kind byte, v any) int64 {
	switch kind {
	case 'V':
		return 0
	case 'L':
		switch x := v.(type) {
		case nil:
			return 0
		case string:
			return newStringHandle(x)
		case Value:
			return dupHandle(x.obj)
		case Objecter:
			return dupHandle(x.JavaObject())
		}
		panic(fmt.Sprintf("jvm: cannot return %T as a Java object", v))
	}

	var temps []int64
	return slot(kind, v, &temps)
}

// compileImpl registers each method. With withSelf, argument 0 is the receiver.
func compileImpl(methods Impl, withSelf bool) (spec string, ids []int64) {
	var lines []string
	for signature, fn := range methods {
		paren := strings.IndexByte(signature, '(')
		if paren <= 0 {
			panic("jvm: method key must look like name(desc)ret, got " + signature)
		}

		params, ret := paramKinds(signature[paren:])
		if withSelf {
			params = "L" + params
		}

		lines = append(lines, signature)
		ids = append(ids, registerCallback(fn, params, ret))
	}
	return strings.Join(lines, "\n"), ids
}

// Proxy implements a Java interface with Go functions. Default methods not listed in methods keep
// their Java behaviour.
func Proxy(iface *Class, methods Impl) Object {
	requireBooted()
	spec, ids := compileImpl(methods, false)

	if len(ids) == 0 {
		panic("jvm: Proxy needs at least one method")
	}

	var out C.i64
	status := C.c_proxy(C.i64(token), C.i64(iface.obj.id()), strAddr(spec), C.i64(len(spec)), ptrAddr(&ids[0]), ptrAddr(&out))
	runtime.KeepAlive(spec)
	runtime.KeepAlive(ids)
	runtime.KeepAlive(iface.obj.r)
	if status != statusOK {
		panic(&Throwable{wrap(int64(out))})
	}
	return wrap(int64(out))
}

// Func implements a single-method Java interface. signature looks like "run()V".
func Func(iface string, signature string, fn Callback) Object {
	return Proxy(FindClass(iface), Impl{signature: fn})
}

// Runnable wraps fn as a java.lang.Runnable.
func Runnable(fn func()) Object {
	return Func("java.lang.Runnable", "run()V", func(*Args) any {
		fn()
		return nil
	})
}

// Supplier wraps fn as a java.util.function.Supplier.
func Supplier(fn func() any) Object {
	return Func("java.util.function.Supplier", "get()Ljava/lang/Object;", func(*Args) any { return fn() })
}

// Extend defines a real Java subclass whose listed methods call Go. In each callback, argument 0 is
// this and the Java arguments follow. Constructors mirror the superclass. If super is an interface,
// the class implements it with a no-argument constructor. Non-abstract overrides also get a
// "super$name" method, which is how Go calls the inherited behaviour; see CallSuper.
func Extend(super *Class, name string, methods Impl) *Class {
	requireBooted()
	spec, ids := compileImpl(methods, true)

	var cbs unsafe.Pointer
	if len(ids) > 0 {
		cbs = ptrAddr(&ids[0])
	}

	var out C.i64
	status := C.c_extend(C.i64(token), C.i64(super.obj.id()), strAddr(name), C.i64(len(name)), strAddr(spec), C.i64(len(spec)), cbs, ptrAddr(&out))
	runtime.KeepAlive(name)
	runtime.KeepAlive(spec)
	runtime.KeepAlive(ids)
	runtime.KeepAlive(super.obj.r)
	if status != statusOK {
		panic(&Throwable{wrap(int64(out))})
	}
	return classFromObject(wrap(int64(out)), super.Name+"$Go$"+name)
}

// CallSuper calls the inherited implementation of a method overridden with Extend.
func (c *Class) CallSuper(self Objecter, name, desc string, args ...any) Value {
	return c.Method("super$"+name, desc).Call(self, args...)
}

// BindHook connects a hook declared in gomod.json to fn. fn receives one argument, the
// com.go.craft.bridge.HookContext. Safe to call from package init.
func BindHook(id string, fn Callback) {
	OnBoot(func() {
		cb := registerCallback(fn, "L", 'V')
		C.c_hook_bind(C.i64(token), strAddr(id), C.i64(len(id)), C.i64(cb))
		runtime.KeepAlive(id)
	})
}
