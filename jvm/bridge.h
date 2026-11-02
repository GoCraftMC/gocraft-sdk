// The function table GoCraft hands to every Go mod. Order and signatures must match
// com.go.craft.bridge.Abi; change GOCRAFT_ABI when they do.
#ifndef GOCRAFT_BRIDGE_H
#define GOCRAFT_BRIDGE_H

#include <stdint.h>

#define GOCRAFT_ABI 1

typedef int64_t i64;

typedef struct gocraft_table {
	int32_t (*find_class)(i64 name, i64 len, i64 out);
	int32_t (*find_method)(i64 cls, i64 name, i64 nlen, i64 desc, i64 dlen, i64 is_static, i64 out);
	int32_t (*find_field)(i64 cls, i64 name, i64 nlen, i64 desc, i64 dlen, i64 is_static, i64 out);
	int32_t (*invoke)(i64 method, i64 recv, i64 args, i64 nargs, i64 out);
	int32_t (*field_get)(i64 field, i64 recv, i64 out);
	int32_t (*field_set)(i64 field, i64 recv, i64 val, i64 out);
	i64 (*string_new)(i64 data, i64 len);
	i64 (*string_read)(i64 h, i64 buf, i64 cap);
	void (*release)(i64 h);
	i64 (*dup)(i64 h);
	i64 (*same)(i64 a, i64 b);
	i64 (*instance_of)(i64 h, i64 cls);
	i64 (*class_of)(i64 h);
	i64 (*array_len)(i64 h);
	i64 (*bytes_new)(i64 data, i64 len);
	i64 (*bytes_read)(i64 h, i64 buf, i64 cap);
	int32_t (*proxy)(i64 token, i64 iface, i64 spec, i64 slen, i64 cbs, i64 out);
	int32_t (*extend)(i64 token, i64 super, i64 name, i64 nlen, i64 spec, i64 slen, i64 cbs, i64 out);
	void (*log)(i64 token, i64 level, i64 data, i64 len);
	void (*hook_bind)(i64 token, i64 id, i64 idlen, i64 cb);
} gocraft_table;

extern const gocraft_table* gocraft_T __attribute__((visibility("hidden")));

#define P(x) ((i64)(intptr_t)(x))

// Pointers cross as void* so that cgo treats them as real pointers: Go then keeps their targets on
// the heap and pinned for the call. A nested Java->Go callback can grow the goroutine's stack, which
// would invalidate any stack address Java was still holding.
static inline int32_t c_find_class(const void* name, i64 len, void* out) { return gocraft_T->find_class(P(name), len, P(out)); }
static inline int32_t c_find_method(i64 cls, const void* name, i64 nlen, const void* desc, i64 dlen, i64 st, void* out) { return gocraft_T->find_method(cls, P(name), nlen, P(desc), dlen, st, P(out)); }
static inline int32_t c_find_field(i64 cls, const void* name, i64 nlen, const void* desc, i64 dlen, i64 st, void* out) { return gocraft_T->find_field(cls, P(name), nlen, P(desc), dlen, st, P(out)); }
static inline int32_t c_invoke(i64 m, i64 recv, const void* args, i64 n, void* out) { return gocraft_T->invoke(m, recv, P(args), n, P(out)); }
static inline int32_t c_field_get(i64 f, i64 recv, void* out) { return gocraft_T->field_get(f, recv, P(out)); }
static inline int32_t c_field_set(i64 f, i64 recv, i64 val, void* out) { return gocraft_T->field_set(f, recv, val, P(out)); }
static inline i64 c_string_new(const void* s, i64 n) { return gocraft_T->string_new(P(s), n); }
static inline i64 c_string_read(i64 h, void* buf, i64 cap) { return gocraft_T->string_read(h, P(buf), cap); }
static inline void c_release(i64 h) { gocraft_T->release(h); }
static inline i64 c_dup(i64 h) { return gocraft_T->dup(h); }
static inline i64 c_same(i64 a, i64 b) { return gocraft_T->same(a, b); }
static inline i64 c_instance_of(i64 h, i64 c) { return gocraft_T->instance_of(h, c); }
static inline i64 c_class_of(i64 h) { return gocraft_T->class_of(h); }
static inline i64 c_array_len(i64 h) { return gocraft_T->array_len(h); }
static inline i64 c_bytes_new(const void* b, i64 n) { return gocraft_T->bytes_new(P(b), n); }
static inline i64 c_bytes_read(i64 h, void* buf, i64 cap) { return gocraft_T->bytes_read(h, P(buf), cap); }
static inline int32_t c_proxy(i64 tok, i64 iface, const void* spec, i64 slen, const void* cbs, void* out) { return gocraft_T->proxy(tok, iface, P(spec), slen, P(cbs), P(out)); }
static inline int32_t c_extend(i64 tok, i64 sup, const void* name, i64 nlen, const void* spec, i64 slen, const void* cbs, void* out) { return gocraft_T->extend(tok, sup, P(name), nlen, P(spec), slen, P(cbs), P(out)); }
static inline void c_log(i64 tok, i64 level, const void* s, i64 n) { gocraft_T->log(tok, level, P(s), n); }
static inline void c_hook_bind(i64 tok, const void* id, i64 n, i64 cb) { gocraft_T->hook_bind(tok, P(id), n, cb); }
static inline void c_set_table(i64 t) { gocraft_T = (const gocraft_table*)(intptr_t)t; }

#endif
