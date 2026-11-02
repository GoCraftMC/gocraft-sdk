// Package event subscribes Go functions to game events, backed by Fabric API's events. Handlers run
// on the thread that fired the event, which is normally the game thread.
package event

import (
	"github.com/GoCraftMC/gocraft-sdk/jvm"
	"github.com/GoCraftMC/gocraft-sdk/mc"
)

const (
	cTick       = "net.fabricmc.fabric.api.event.lifecycle.v1.ServerTickEvents"
	cLifecycle  = "net.fabricmc.fabric.api.event.lifecycle.v1.ServerLifecycleEvents"
	cConnection = "net.fabricmc.fabric.api.networking.v1.ServerPlayConnectionEvents"
	cBreak      = "net.fabricmc.fabric.api.event.player.PlayerBlockBreakEvents"
	cUseBlock   = "net.fabricmc.fabric.api.event.player.UseBlockCallback"
	cUseItem    = "net.fabricmc.fabric.api.event.player.UseItemCallback"
	cClientTick = "net.fabricmc.fabric.api.client.event.lifecycle.v1.ClientTickEvents"
	cListener   = "net.minecraft.server.network.ServerGamePacketListenerImpl"

	descEvent  = "Lnet/fabricmc/fabric/api/event/Event;"
	descServer = "Lnet/minecraft/server/MinecraftServer;"
	descResult = "Lnet/minecraft/world/InteractionResult;"
	descPlayer = "Lnet/minecraft/world/entity/player/Player;"
	descLevel  = "Lnet/minecraft/world/level/Level;"
	descHand   = "Lnet/minecraft/world/InteractionHand;"
)

var (
	eventRegister = jvm.M("net.fabricmc.fabric.api.event.Event", "register", "(Ljava/lang/Object;)V")
	listenerGet   = jvm.M(cListener, "getPlayer", "()Lnet/minecraft/server/level/ServerPlayer;")
	hitBlockPos   = jvm.M("net.minecraft.world.phys.BlockHitResult", "getBlockPos", "()Lnet/minecraft/core/BlockPos;")
)

func listen(class, field, iface, signature string, fn jvm.Callback) {
	event := jvm.FindClass(class).StaticField(field, descEvent).GetStatic()
	eventRegister.Call(event, jvm.Func(iface, signature, fn))
}

func serverEvent(class, field, nested, method string, fn func(mc.Server)) {
	listen(class, field, class+"$"+nested, method+"("+descServer+")V", func(a *jvm.Args) any {
		fn(mc.Server{Object: a.Object(0)})
		return nil
	})
}

// ServerTick runs fn at the end of every server tick (20 times a second).
func ServerTick(fn func(mc.Server)) {
	serverEvent(cTick, "END_SERVER_TICK", "EndTick", "onEndTick", fn)
}

// ServerTickStart runs fn at the start of every server tick.
func ServerTickStart(fn func(mc.Server)) {
	serverEvent(cTick, "START_SERVER_TICK", "StartTick", "onStartTick", fn)
}

// ServerStarting runs fn while the server is starting, before worlds load.
func ServerStarting(fn func(mc.Server)) {
	serverEvent(cLifecycle, "SERVER_STARTING", "ServerStarting", "onServerStarting", fn)
}

// ServerStarted runs fn once the server is ready for players.
func ServerStarted(fn func(mc.Server)) {
	serverEvent(cLifecycle, "SERVER_STARTED", "ServerStarted", "onServerStarted", fn)
}

// ServerStopping runs fn when the server begins shutting down.
func ServerStopping(fn func(mc.Server)) {
	serverEvent(cLifecycle, "SERVER_STOPPING", "ServerStopping", "onServerStopping", fn)
}

// PlayerJoin runs fn when a player has joined and can receive packets.
func PlayerJoin(fn func(mc.Player, mc.Server)) {
	listen(cConnection, "JOIN", cConnection+"$Join",
		"onPlayReady(L"+slashes(cListener)+";Lnet/fabricmc/fabric/api/networking/v1/PacketSender;"+descServer+")V",
		func(a *jvm.Args) any {
			fn(mc.Player{Object: listenerGet.Call(a.Object(0)).Object()}, mc.Server{Object: a.Object(2)})
			return nil
		})
}

// PlayerLeave runs fn when a player disconnects.
func PlayerLeave(fn func(mc.Player, mc.Server)) {
	listen(cConnection, "DISCONNECT", cConnection+"$Disconnect",
		"onPlayDisconnect(L"+slashes(cListener)+";"+descServer+")V",
		func(a *jvm.Args) any {
			fn(mc.Player{Object: listenerGet.Call(a.Object(0)).Object()}, mc.Server{Object: a.Object(1)})
			return nil
		})
}

// BlockBroken runs fn after a player breaks a block.
func BlockBroken(fn func(level mc.Level, player mc.Player, pos mc.BlockPos, state mc.BlockState)) {
	listen(cBreak, "AFTER", cBreak+"$After",
		"afterBlockBreak("+descLevel+descPlayer+"Lnet/minecraft/core/BlockPos;Lnet/minecraft/world/level/block/state/BlockState;Lnet/minecraft/world/level/block/entity/BlockEntity;)V",
		func(a *jvm.Args) any {
			fn(mc.Level{Object: a.Object(0)}, mc.Player{Object: a.Object(1)}, mc.BlockPos{Object: a.Object(2)}, mc.BlockState{Object: a.Object(3)})
			return nil
		})
}

// UseBlock runs fn when a player right-clicks a block. Return mc.Pass to let the game continue.
func UseBlock(fn func(player mc.Player, level mc.Level, pos mc.BlockPos) mc.InteractionResult) {
	listen(cUseBlock, "EVENT", cUseBlock,
		"interact("+descPlayer+descLevel+descHand+"Lnet/minecraft/world/phys/BlockHitResult;)"+descResult,
		func(a *jvm.Args) any {
			pos := mc.BlockPos{Object: hitBlockPos.Call(a.Object(3)).Object()}
			return fn(mc.Player{Object: a.Object(0)}, mc.Level{Object: a.Object(1)}, pos)
		})
}

// UseItem runs fn when a player right-clicks with an item. Return mc.Pass to let the game continue.
func UseItem(fn func(player mc.Player, level mc.Level) mc.InteractionResult) {
	listen(cUseItem, "EVENT", cUseItem,
		"interact("+descPlayer+descLevel+descHand+")"+descResult,
		func(a *jvm.Args) any { return fn(mc.Player{Object: a.Object(0)}, mc.Level{Object: a.Object(1)}) })
}

// ClientTick runs fn at the end of every client tick. The argument is the Minecraft client object.
// Register it from gocraft.OnInitializeClient: the event does not exist on a dedicated server.
func ClientTick(fn func(client jvm.Object)) {
	listen(cClientTick, "END_CLIENT_TICK", cClientTick+"$EndTick", "onEndTick(Lnet/minecraft/client/Minecraft;)V",
		func(a *jvm.Args) any {
			fn(a.Object(0))
			return nil
		})
}

func slashes(class string) string {
	b := []byte(class)
	for i, c := range b {
		if c == '.' {
			b[i] = '/'
		}
	}
	return string(b)
}
