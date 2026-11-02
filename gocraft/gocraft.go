// Package gocraft is the entry point of a Go mod: lifecycle entrypoints, logging, hooks and config.
//
// A mod's entrypoint packages register themselves from init, as go.jar links in the packages listed
// under "entrypoints" in gomod.json:
//
//	func init() {
//		gocraft.OnInitialize(func() {
//			gocraft.Log().Info("hello from Go")
//		})
//	}
package gocraft

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"

	"github.com/GoCraftMC/gocraft-sdk/jvm"
)

// OnPreLaunch runs fn before Minecraft's classes load. Keep it small: nothing from the game is
// usable yet. The equivalent of Fabric's preLaunch entrypoint.
func OnPreLaunch(fn func()) { jvm.OnEntry("preLaunch", fn) }

// OnInitialize runs fn on both client and server when mods initialize, which is the time to register
// content. The equivalent of Fabric's ModInitializer.
func OnInitialize(fn func()) { jvm.OnEntry("main", fn) }

// OnInitializeClient runs fn on the physical client only.
func OnInitializeClient(fn func()) { jvm.OnEntry("client", fn) }

// OnInitializeServer runs fn on a dedicated server only.
func OnInitializeServer(fn func()) { jvm.OnEntry("server", fn) }

// Entrypoint runs fn for a custom entrypoint key, listed under "customEntrypoints" in gomod.json.
func Entrypoint(key string, fn func()) { jvm.OnEntry(key, fn) }

// ModID returns this mod's id from gomod.json.
func ModID() string { return jvm.ModID() }

type gameHandler struct {
	attrs []slog.Attr
	group string
}

func (h gameHandler) Enabled(context.Context, slog.Level) bool { return true }

func (h gameHandler) Handle(_ context.Context, r slog.Record) error {
	var b strings.Builder
	b.WriteString(r.Message)

	write := func(a slog.Attr) bool {
		b.WriteByte(' ')
		if h.group != "" {
			b.WriteString(h.group + ".")
		}
		fmt.Fprintf(&b, "%s=%v", a.Key, a.Value)
		return true
	}
	for _, a := range h.attrs {
		write(a)
	}
	r.Attrs(write)

	level := jvm.LevelInfo
	switch {
	case r.Level >= slog.LevelError:
		level = jvm.LevelError
	case r.Level >= slog.LevelWarn:
		level = jvm.LevelWarn
	case r.Level < slog.LevelInfo:
		level = jvm.LevelDebug
	}
	jvm.Log(level, b.String())
	return nil
}

func (h gameHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	h.attrs = append(h.attrs[:len(h.attrs):len(h.attrs)], attrs...)
	return h
}

func (h gameHandler) WithGroup(name string) slog.Handler {
	if h.group != "" {
		name = h.group + "." + name
	}
	h.group = name
	return h
}

var logger = slog.New(gameHandler{})

func init() {
	// Plain log and slog calls, including those made by libraries, land in the game log.
	jvm.OnBoot(func() { slog.SetDefault(logger) })
}

// Log returns a logger that writes to the game log under this mod's name.
func Log() *slog.Logger { return logger }

// Go starts a goroutine whose panics are logged. An unrecovered panic in a plain goroutine ends the
// whole process, including the game, because a Go mod shares the JVM's process.
func Go(fn func()) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				logger.Error(fmt.Sprintf("panic in background goroutine: %v\n%s", r, debug.Stack()))
			}
		}()
		fn()
	}()
}

var (
	fabricLoaderInstance = jvm.SM("net.fabricmc.loader.api.FabricLoader", "getInstance", "()Lnet/fabricmc/loader/api/FabricLoader;")
	loaderConfigDir      = jvm.M("net.fabricmc.loader.api.FabricLoader", "getConfigDir", "()Ljava/nio/file/Path;")
	loaderGameDir        = jvm.M("net.fabricmc.loader.api.FabricLoader", "getGameDir", "()Ljava/nio/file/Path;")
	loaderEnvType        = jvm.M("net.fabricmc.loader.api.FabricLoader", "getEnvironmentType", "()Lnet/fabricmc/api/EnvType;")
	loaderIsModLoaded    = jvm.M("net.fabricmc.loader.api.FabricLoader", "isModLoaded", "(Ljava/lang/String;)Z")
)

// ConfigDir returns the game's config directory.
func ConfigDir() string { return loaderConfigDir.Call(fabricLoaderInstance.CallStatic()).String() }

// GameDir returns the game's run directory.
func GameDir() string { return loaderGameDir.Call(fabricLoaderInstance.CallStatic()).String() }

// IsClient reports whether this is the physical client rather than a dedicated server.
func IsClient() bool {
	return loaderEnvType.Call(fabricLoaderInstance.CallStatic()).String() == "CLIENT"
}

// IsModLoaded reports whether a Go or Fabric mod with this id is present.
func IsModLoaded(id string) bool {
	return loaderIsModLoaded.Call(fabricLoaderInstance.CallStatic(), id).Bool()
}

// LoadConfig reads config/<modid>.json into cfg. If the file is missing, it is created from cfg's
// current values, so set the defaults before calling.
func LoadConfig(cfg any) error {
	path := filepath.Join(ConfigDir(), ModID()+".json")

	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		data, err = json.MarshalIndent(cfg, "", "  ")
		if err != nil {
			return err
		}
		return os.WriteFile(path, append(data, '\n'), 0o644)
	}
	if err != nil {
		return err
	}
	return json.Unmarshal(data, cfg)
}

const hookContextClass = "com.go.craft.bridge.HookContext"

var (
	hookSelf     = jvm.M(hookContextClass, "self", "()Ljava/lang/Object;")
	hookArg      = jvm.M(hookContextClass, "arg", "(I)Ljava/lang/Object;")
	hookArgCount = jvm.M(hookContextClass, "argCount", "()I")
	hookCancel   = jvm.M(hookContextClass, "cancel", "()V")
	hookReturn   = jvm.M(hookContextClass, "returnValue", "()Ljava/lang/Object;")
	hookSet      = jvm.M(hookContextClass, "setReturn", "(Ljava/lang/Object;)V")
)

// HookContext describes one firing of a hook declared in gomod.json.
type HookContext struct{ obj jvm.Object }

// Self returns the object the target method ran on. It is null for static targets.
func (h *HookContext) Self() jvm.Object { return hookSelf.Call(h.obj).Object() }

// Arg returns argument i of the target method. Primitives arrive boxed.
func (h *HookContext) Arg(i int) jvm.Object { return hookArg.Call(h.obj, i).Object() }

// ArgCount returns how many arguments the target method has.
func (h *HookContext) ArgCount() int { return int(hookArgCount.Call(h.obj).Int()) }

// Cancel stops the target method. Unavailable on TAIL hooks.
func (h *HookContext) Cancel() { hookCancel.Call(h.obj) }

// ReturnValue reads the value being returned, at a RETURN hook on a non-void method.
func (h *HookContext) ReturnValue() jvm.Object { return hookReturn.Call(h.obj).Object() }

// SetReturn replaces the return value and leaves the target method. Primitives must be boxed.
func (h *HookContext) SetReturn(value any) { hookSet.Call(h.obj, value) }

// Hook binds fn to a hook declared under "hooks" in gomod.json. Safe to call from init, and must be
// called early, since the hook can fire as soon as the target class runs.
func Hook(id string, fn func(*HookContext)) {
	jvm.BindHook(id, func(a *jvm.Args) any {
		fn(&HookContext{obj: a.Object(0)})
		return nil
	})
}
