package jvm

// #include "bridge.h"
import "C"

import (
	"fmt"
	"math"
	"runtime"
	"strings"
	"sync"
	"unsafe"
)

// Throwable is a Java exception surfaced in Go. Bridge calls panic with it, because any Java call
// can throw; catch one with Try. If it goes uncaught back into Java, Java sees the original.
type Throwable struct{ Object }

func (t *Throwable) Error() string { return t.Object.String() }

// Try runs fn and returns any Java exception it raised. Other panics keep unwinding.
func Try(fn func()) (err error) {
	defer func() {
		if r := recover(); r != nil {
			thrown, ok := r.(*Throwable)
			if !ok {
				panic(r)
			}
			err = thrown
		}
	}()
	fn()
	return nil
}

// Class is a loaded Java class.
type Class struct {
	obj  Object
	Name string
}

// JavaObject implements Objecter, giving access to the java.lang.Class instance.
func (c *Class) JavaObject() Object { return c.obj }

var classes sync.Map

// FindClass loads a class by binary name, such as "net.minecraft.world.item.Item" or
// "java.util.Map$Entry". It panics with a Throwable if the class does not exist.
func FindClass(name string) *Class {
	requireBooted()
	if c, ok := classes.Load(name); ok {
		return c.(*Class)
	}

	var out C.i64
	status := C.c_find_class(strAddr(name), C.i64(len(name)), ptrAddr(&out))
	runtime.KeepAlive(name)
	if status != statusOK {
		panic(&Throwable{wrap(int64(out))})
	}

	actual, _ := classes.LoadOrStore(name, &Class{obj: wrap(int64(out)), Name: name})
	return actual.(*Class)
}

func classFromObject(o Object, name string) *Class { return &Class{obj: o, Name: name} }

// Method is a resolved Java method or constructor.
type Method struct {
	id     int64
	params string
	ret    byte
	static bool
	label  string
}

// Field is a resolved Java field.
type Field struct {
	id     int64
	kind   byte
	static bool
}

// paramKinds parses "(ILjava/lang/String;[B)V" into per-parameter kinds "ILL" and return kind 'V'.
func paramKinds(desc string) (params string, ret byte) {
	if !strings.HasPrefix(desc, "(") {
		panic("jvm: malformed descriptor " + desc)
	}

	var kinds []byte
	i := 1
	for desc[i] != ')' {
		start := desc[i]
		for desc[i] == '[' {
			i++
		}
		if desc[i] == 'L' {
			i += strings.IndexByte(desc[i:], ';')
		}
		i++
		if start == '[' || start == 'L' {
			kinds = append(kinds, 'L')
		} else {
			kinds = append(kinds, start)
		}
	}

	ret = desc[i+1]
	if ret == '[' {
		ret = 'L'
	}
	return string(kinds), ret
}

func (c *Class) findMethod(name, desc string, static bool) *Method {
	var out C.i64
	isStatic := C.i64(0)
	if static {
		isStatic = 1
	}

	status := C.c_find_method(C.i64(c.obj.id()), strAddr(name), C.i64(len(name)), strAddr(desc), C.i64(len(desc)), isStatic, ptrAddr(&out))
	runtime.KeepAlive(name)
	runtime.KeepAlive(desc)
	runtime.KeepAlive(c.obj.r)
	if status != statusOK {
		panic(&Throwable{wrap(int64(out))})
	}

	params, ret := paramKinds(desc)
	if name == "<init>" {
		ret = 'L'
	}
	return &Method{id: int64(out), params: params, ret: ret, static: static || name == "<init>", label: c.Name + "." + name + desc}
}

// Method resolves an instance method, including private and inherited ones.
func (c *Class) Method(name, desc string) *Method { return c.findMethod(name, desc, false) }

// StaticMethod resolves a static method.
func (c *Class) StaticMethod(name, desc string) *Method { return c.findMethod(name, desc, true) }

// Constructor resolves a constructor, for example "(I)V".
func (c *Class) Constructor(desc string) *Method { return c.findMethod("<init>", desc, true) }

// New constructs an instance using the constructor with the given descriptor.
func (c *Class) New(desc string, args ...any) Object {
	return c.Constructor(desc).CallStatic(args...).Object()
}

func (c *Class) findField(name, desc string, static bool) *Field {
	var out C.i64
	isStatic := C.i64(0)
	if static {
		isStatic = 1
	}

	status := C.c_find_field(C.i64(c.obj.id()), strAddr(name), C.i64(len(name)), strAddr(desc), C.i64(len(desc)), isStatic, ptrAddr(&out))
	runtime.KeepAlive(name)
	runtime.KeepAlive(desc)
	runtime.KeepAlive(c.obj.r)
	if status != statusOK {
		panic(&Throwable{wrap(int64(out))})
	}

	kind := desc[0]
	if kind == '[' {
		kind = 'L'
	}
	return &Field{id: int64(out), kind: kind, static: static}
}

// Field resolves an instance field, including private ones.
func (c *Class) Field(name, desc string) *Field { return c.findField(name, desc, false) }

// StaticField resolves a static field.
func (c *Class) StaticField(name, desc string) *Field { return c.findField(name, desc, true) }

// Value is a Java primitive or reference returned from a call or field read.
type Value struct {
	kind byte
	bits int64
	obj  Object
}

func valueOf(kind byte, bits int64) Value {
	if kind == 'L' {
		return Value{kind: kind, obj: wrap(bits)}
	}
	return Value{kind: kind, bits: bits}
}

// Object returns the reference held by v.
func (v Value) Object() Object { return v.obj }

// JavaObject implements Objecter, so a call's result can be passed straight to another call.
func (v Value) JavaObject() Object { return v.obj }

// String returns the Java string held by v, or its toString().
func (v Value) String() string { return v.obj.String() }

// Int returns v as an int32.
func (v Value) Int() int32 { return int32(v.bits) }

// Long returns v as an int64.
func (v Value) Long() int64 { return v.bits }

// Bool returns v as a bool.
func (v Value) Bool() bool { return v.bits != 0 }

// Float returns v as a float32.
func (v Value) Float() float32 { return math.Float32frombits(uint32(v.bits)) }

// Double returns v as a float64.
func (v Value) Double() float64 { return math.Float64frombits(uint64(v.bits)) }

// slot converts a Go value into the raw form for a parameter of the given kind. Strings allocate a
// Java String, whose handle is appended to temps for the caller to release.
func slot(kind byte, v any, temps *[]int64) int64 {
	switch kind {
	case 'L':
		switch x := v.(type) {
		case nil:
			return 0
		case string:
			id := newStringHandle(x)
			*temps = append(*temps, id)
			return id
		case Value:
			return x.obj.id()
		case Objecter:
			return x.JavaObject().id()
		}
		panic(fmt.Sprintf("jvm: cannot pass %T as a Java object", v))
	case 'F':
		return int64(math.Float32bits(float32(number(v))))
	case 'D':
		return int64(math.Float64bits(number(v)))
	}

	switch x := v.(type) {
	case bool:
		if x {
			return 1
		}
		return 0
	case int:
		return int64(x)
	case int8:
		return int64(x)
	case int16:
		return int64(x)
	case int32:
		return int64(x)
	case int64:
		return x
	case uint8:
		return int64(x)
	case uint16:
		return int64(x)
	case uint32:
		return int64(x)
	case Value:
		return x.bits
	}
	panic(fmt.Sprintf("jvm: cannot pass %T as Java kind %c", v, kind))
}

func number(v any) float64 {
	switch x := v.(type) {
	case float64:
		return x
	case float32:
		return float64(x)
	case int:
		return float64(x)
	case int32:
		return float64(x)
	case int64:
		return float64(x)
	case Value:
		if x.kind == 'F' {
			return float64(x.Float())
		}
		return x.Double()
	}
	panic(fmt.Sprintf("jvm: cannot pass %T as a Java float or double", v))
}

func (m *Method) invoke(recv Object, args []any) Value {
	if len(args) != len(m.params) {
		panic(fmt.Sprintf("jvm: %s takes %d arguments, got %d", m.label, len(m.params), len(args)))
	}

	slots := make([]int64, len(args))

	var temps []int64
	for i, arg := range args {
		slots[i] = slot(m.params[i], arg, &temps)
	}

	var argsAddr unsafe.Pointer
	if len(slots) > 0 {
		argsAddr = ptrAddr(&slots[0])
	}

	var out C.i64
	status := C.c_invoke(C.i64(m.id), C.i64(recv.id()), argsAddr, C.i64(len(slots)), ptrAddr(&out))

	for _, id := range temps {
		C.c_release(C.i64(id))
	}
	runtime.KeepAlive(slots)
	runtime.KeepAlive(args)
	runtime.KeepAlive(recv.r)

	if status != statusOK {
		panic(&Throwable{wrap(int64(out))})
	}
	return valueOf(m.ret, int64(out))
}

// Call invokes an instance method on recv.
func (m *Method) Call(recv Objecter, args ...any) Value { return m.invoke(recv.JavaObject(), args) }

// CallStatic invokes a static method or constructor.
func (m *Method) CallStatic(args ...any) Value { return m.invoke(Object{}, args) }

// Get reads the field on recv.
func (f *Field) Get(recv Objecter) Value {
	obj := recv.JavaObject()
	var out C.i64
	status := C.c_field_get(C.i64(f.id), C.i64(obj.id()), ptrAddr(&out))
	runtime.KeepAlive(obj.r)
	if status != statusOK {
		panic(&Throwable{wrap(int64(out))})
	}
	return valueOf(f.kind, int64(out))
}

// GetStatic reads a static field.
func (f *Field) GetStatic() Value { return f.Get(Object{}) }

// Set writes the field on recv.
func (f *Field) Set(recv Objecter, value any) {
	obj := recv.JavaObject()
	var temps []int64
	raw := slot(f.kind, value, &temps)

	var out C.i64
	status := C.c_field_set(C.i64(f.id), C.i64(obj.id()), C.i64(raw), ptrAddr(&out))
	for _, id := range temps {
		C.c_release(C.i64(id))
	}
	runtime.KeepAlive(value)
	runtime.KeepAlive(obj.r)
	if status != statusOK {
		panic(&Throwable{wrap(int64(out))})
	}
}

// SetStatic writes a static field.
func (f *Field) SetStatic(value any) { f.Set(Object{}, value) }

// LazyMethod resolves on first use, so a package var can name a client-only class without touching
// it on a dedicated server.
type LazyMethod struct {
	once              sync.Once
	class, name, desc string
	static            bool
	method            *Method
}

// M declares a lazily resolved instance method.
func M(class, name, desc string) *LazyMethod {
	return &LazyMethod{class: class, name: name, desc: desc}
}

// SM declares a lazily resolved static method.
func SM(class, name, desc string) *LazyMethod {
	return &LazyMethod{class: class, name: name, desc: desc, static: true}
}

// Ctor declares a lazily resolved constructor.
func Ctor(class, desc string) *LazyMethod {
	return &LazyMethod{class: class, name: "<init>", desc: desc, static: true}
}

func (l *LazyMethod) get() *Method {
	l.once.Do(func() { l.method = FindClass(l.class).findMethod(l.name, l.desc, l.static) })
	return l.method
}

// Call invokes the method on recv.
func (l *LazyMethod) Call(recv Objecter, args ...any) Value { return l.get().Call(recv, args...) }

// CallStatic invokes the static method or constructor.
func (l *LazyMethod) CallStatic(args ...any) Value { return l.get().CallStatic(args...) }

// LazyField resolves on first use.
type LazyField struct {
	once              sync.Once
	class, name, desc string
	static            bool
	field             *Field
}

// F declares a lazily resolved instance field.
func F(class, name, desc string) *LazyField { return &LazyField{class: class, name: name, desc: desc} }

// SF declares a lazily resolved static field.
func SF(class, name, desc string) *LazyField {
	return &LazyField{class: class, name: name, desc: desc, static: true}
}

func (l *LazyField) get() *Field {
	l.once.Do(func() { l.field = FindClass(l.class).findField(l.name, l.desc, l.static) })
	return l.field
}

// Get reads the field on recv.
func (l *LazyField) Get(recv Objecter) Value { return l.get().Get(recv) }

// GetStatic reads a static field.
func (l *LazyField) GetStatic() Value { return l.get().GetStatic() }

// Set writes the field on recv.
func (l *LazyField) Set(recv Objecter, value any) { l.get().Set(recv, value) }

// SetStatic writes a static field.
func (l *LazyField) SetStatic(value any) { l.get().SetStatic(value) }
