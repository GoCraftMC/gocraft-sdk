// Package mc wraps the parts of Minecraft that nearly every mod touches, with Go-style names. For
// everything else, go.jar's genBindings task generates typed bindings for any class you list, and
// package jvm reaches any member by name.
package mc

import (
	"strings"

	"github.com/GoCraftMC/gocraft-sdk/jvm"
)

const (
	cIdentifier = "net.minecraft.resources.Identifier"
	cComponent  = "net.minecraft.network.chat.Component"
	cServer     = "net.minecraft.server.MinecraftServer"
	cPlayerList = "net.minecraft.server.players.PlayerList"
	cEntity     = "net.minecraft.world.entity.Entity"
	cLiving     = "net.minecraft.world.entity.LivingEntity"
	cPlayer     = "net.minecraft.world.entity.player.Player"
	cLevel      = "net.minecraft.world.level.Level"
	cBlockPos   = "net.minecraft.core.BlockPos"
	cVec3i      = "net.minecraft.core.Vec3i"
	cStack      = "net.minecraft.world.item.ItemStack"
	cState      = "net.minecraft.world.level.block.state.BlockBehaviour$BlockStateBase"
	cBlock      = "net.minecraft.world.level.block.Block"
	cResult     = "net.minecraft.world.InteractionResult"
)

// Identifier is a namespaced id such as minecraft:stone.
type Identifier struct{ jvm.Object }

var (
	idFromParts = jvm.SM(cIdentifier, "fromNamespaceAndPath", "(Ljava/lang/String;Ljava/lang/String;)Lnet/minecraft/resources/Identifier;")
	idParse     = jvm.SM(cIdentifier, "parse", "(Ljava/lang/String;)Lnet/minecraft/resources/Identifier;")
)

// ID builds an Identifier. "stone" uses this mod's namespace; "minecraft:stone" is taken as written.
func ID(id string) Identifier {
	if strings.Contains(id, ":") {
		return Identifier{idParse.CallStatic(id).Object()}
	}
	return Identifier{idFromParts.CallStatic(jvm.ModID(), id).Object()}
}

// Component is rich chat text.
type Component struct{ jvm.Object }

var (
	componentLiteral      = jvm.SM(cComponent, "literal", "(Ljava/lang/String;)Lnet/minecraft/network/chat/MutableComponent;")
	componentTranslatable = jvm.SM(cComponent, "translatable", "(Ljava/lang/String;)Lnet/minecraft/network/chat/MutableComponent;")
	componentGetString    = jvm.M(cComponent, "getString", "()Ljava/lang/String;")
)

// Text makes a literal Component.
func Text(s string) Component { return Component{componentLiteral.CallStatic(s).Object()} }

// Translatable makes a Component resolved from language files.
func Translatable(key string) Component {
	return Component{componentTranslatable.CallStatic(key).Object()}
}

// Plain returns the component as unformatted text.
func (c Component) Plain() string { return componentGetString.Call(c).String() }

// Server is the running MinecraftServer.
type Server struct{ jvm.Object }

var (
	serverPlayerList = jvm.M(cServer, "getPlayerList", "()Lnet/minecraft/server/players/PlayerList;")
	serverTick       = jvm.M(cServer, "getTickCount", "()I")
	serverOverworld  = jvm.M(cServer, "overworld", "()Lnet/minecraft/server/level/ServerLevel;")
	serverExecute    = jvm.M("net.minecraft.util.thread.BlockableEventLoop", "execute", "(Ljava/lang/Runnable;)V")
	serverVersion    = jvm.M(cServer, "getServerVersion", "()Ljava/lang/String;")
	serverCommands   = jvm.M(cServer, "getCommands", "()Lnet/minecraft/commands/Commands;")
	serverSource     = jvm.M(cServer, "createCommandSourceStack", "()Lnet/minecraft/commands/CommandSourceStack;")
	commandsPerform  = jvm.M("net.minecraft.commands.Commands", "performPrefixedCommand", "(Lnet/minecraft/commands/CommandSourceStack;Ljava/lang/String;)V")
	listPlayers      = jvm.M(cPlayerList, "getPlayers", "()Ljava/util/List;")
	listBroadcast    = jvm.M(cPlayerList, "broadcastSystemMessage", "(Lnet/minecraft/network/chat/Component;Z)V")
	listCount        = jvm.M(cPlayerList, "getPlayerCount", "()I")
	javaListSize     = jvm.M("java.util.List", "size", "()I")
	javaListGet      = jvm.M("java.util.List", "get", "(I)Ljava/lang/Object;")
)

// Tick returns the number of ticks since the server started.
func (s Server) Tick() int { return int(serverTick.Call(s).Int()) }

// Overworld returns the main dimension.
func (s Server) Overworld() Level { return Level{serverOverworld.Call(s).Object()} }

// Version returns the game version the server runs.
func (s Server) Version() string { return serverVersion.Call(s).String() }

// PlayerCount returns how many players are online.
func (s Server) PlayerCount() int {
	return int(listCount.Call(serverPlayerList.Call(s)).Int())
}

// Players returns the online players.
func (s Server) Players() []Player {
	list := listPlayers.Call(serverPlayerList.Call(s))
	players := make([]Player, javaListSize.Call(list).Int())
	for i := range players {
		players[i] = Player{javaListGet.Call(list, i).Object()}
	}
	return players
}

// Broadcast sends a system message to every player and the console.
func (s Server) Broadcast(message string) {
	listBroadcast.Call(serverPlayerList.Call(s), Text(message), false)
}

// Execute queues fn on the server thread. Use it from goroutines: game state may only be touched
// from the server thread.
func (s Server) Execute(fn func()) { serverExecute.Call(s, jvm.Runnable(fn)) }

// RunCommand runs a command with full permissions, as the server console would.
func (s Server) RunCommand(command string) {
	commandsPerform.Call(serverCommands.Call(s), serverSource.Call(s), command)
}

// Entity is any entity in the world.
type Entity struct{ jvm.Object }

var (
	entityBlockPos = jvm.M(cEntity, "blockPosition", "()Lnet/minecraft/core/BlockPos;")
	entityLevel    = jvm.M(cEntity, "level", "()Lnet/minecraft/world/level/Level;")
	entityUUID     = jvm.M(cEntity, "getStringUUID", "()Ljava/lang/String;")
	entityX        = jvm.M(cEntity, "getX", "()D")
	entityY        = jvm.M(cEntity, "getY", "()D")
	entityZ        = jvm.M(cEntity, "getZ", "()D")
	entityName     = jvm.M(cEntity, "getName", "()Lnet/minecraft/network/chat/Component;")
)

// BlockPos returns the block the entity occupies.
func (e Entity) BlockPos() BlockPos { return BlockPos{entityBlockPos.Call(e).Object()} }

// Level returns the world the entity is in.
func (e Entity) Level() Level { return Level{entityLevel.Call(e).Object()} }

// UUID returns the entity's UUID as a string.
func (e Entity) UUID() string { return entityUUID.Call(e).String() }

// Position returns the exact coordinates.
func (e Entity) Position() (x, y, z float64) {
	return entityX.Call(e).Double(), entityY.Call(e).Double(), entityZ.Call(e).Double()
}

// Name returns the display name as plain text.
func (e Entity) Name() string { return Component{entityName.Call(e).Object()}.Plain() }

// Player is a player, on either side.
type Player struct{ jvm.Object }

var (
	playerSend      = jvm.M(cPlayer, "sendSystemMessage", "(Lnet/minecraft/network/chat/Component;)V")
	playerAddItem   = jvm.M(cPlayer, "addItem", "(Lnet/minecraft/world/item/ItemStack;)Z")
	livingHealth    = jvm.M(cLiving, "getHealth", "()F")
	livingSetHealth = jvm.M(cLiving, "setHealth", "(F)V")
	playerName      = jvm.M(cPlayer, "getScoreboardName", "()Ljava/lang/String;")
)

// Entity returns the player as an Entity.
func (p Player) Entity() Entity { return Entity{p.Object} }

// Name returns the player's username.
func (p Player) Name() string { return playerName.Call(p).String() }

// Send shows a message in the player's chat.
func (p Player) Send(message string) { playerSend.Call(p, Text(message)) }

// Give adds a stack to the inventory and reports whether all of it fit.
func (p Player) Give(stack ItemStack) bool { return playerAddItem.Call(p, stack).Bool() }

// Health returns current health points.
func (p Player) Health() float32 { return livingHealth.Call(p).Float() }

// SetHealth sets current health points.
func (p Player) SetHealth(health float32) { livingSetHealth.Call(p, health) }

// BlockPos is a block coordinate.
type BlockPos struct{ jvm.Object }

var (
	posNew   = jvm.Ctor(cBlockPos, "(III)V")
	posX     = jvm.M(cVec3i, "getX", "()I")
	posY     = jvm.M(cVec3i, "getY", "()I")
	posZ     = jvm.M(cVec3i, "getZ", "()I")
	posAbove = jvm.M(cBlockPos, "above", "()Lnet/minecraft/core/BlockPos;")
)

// Pos makes a BlockPos.
func Pos(x, y, z int) BlockPos { return BlockPos{posNew.CallStatic(x, y, z).Object()} }

// X returns the x coordinate.
func (p BlockPos) X() int { return int(posX.Call(p).Int()) }

// Y returns the y coordinate.
func (p BlockPos) Y() int { return int(posY.Call(p).Int()) }

// Z returns the z coordinate.
func (p BlockPos) Z() int { return int(posZ.Call(p).Int()) }

// Above returns the position one block up.
func (p BlockPos) Above() BlockPos { return BlockPos{posAbove.Call(p).Object()} }

// Level is a world: the overworld, the nether, the end or a custom dimension.
type Level struct{ jvm.Object }

var (
	levelIsClient = jvm.M(cLevel, "isClientSide", "()Z")
	levelGetState = jvm.M(cLevel, "getBlockState", "(Lnet/minecraft/core/BlockPos;)Lnet/minecraft/world/level/block/state/BlockState;")
	levelSetBlock = jvm.M("net.minecraft.world.level.LevelWriter", "setBlockAndUpdate", "(Lnet/minecraft/core/BlockPos;Lnet/minecraft/world/level/block/state/BlockState;)Z")
	levelGameTime = jvm.M(cLevel, "getGameTime", "()J")
	stateGetBlock = jvm.M(cState, "getBlock", "()Lnet/minecraft/world/level/block/Block;")
	blockDefault  = jvm.M(cBlock, "defaultBlockState", "()Lnet/minecraft/world/level/block/state/BlockState;")
)

// IsClient reports whether this is the client's copy of the world.
func (l Level) IsClient() bool { return levelIsClient.Call(l).Bool() }

// GameTime returns the world's total age in ticks.
func (l Level) GameTime() int64 { return levelGameTime.Call(l).Long() }

// BlockState returns the block state at pos.
func (l Level) BlockState(pos BlockPos) BlockState {
	return BlockState{levelGetState.Call(l, pos).Object()}
}

// SetBlock replaces the block at pos and updates neighbours.
func (l Level) SetBlock(pos BlockPos, state BlockState) bool {
	return levelSetBlock.Call(l, pos, state).Bool()
}

// BlockState is a block plus its properties.
type BlockState struct{ jvm.Object }

// Block returns the state's block.
func (s BlockState) Block() Block { return Block{stateGetBlock.Call(s).Object()} }

// Block is a block type.
type Block struct{ jvm.Object }

// DefaultState returns the block's default state.
func (b Block) DefaultState() BlockState { return BlockState{blockDefault.Call(b).Object()} }

// Item is an item type.
type Item struct{ jvm.Object }

// ItemStack is a stack of items.
type ItemStack struct{ jvm.Object }

var (
	stackNew       = jvm.Ctor(cStack, "(Lnet/minecraft/world/level/ItemLike;I)V")
	stackCount     = jvm.M(cStack, "getCount", "()I")
	stackHoverName = jvm.M(cStack, "getHoverName", "()Lnet/minecraft/network/chat/Component;")
)

// Stack makes an ItemStack from an Item or Block.
func Stack(item jvm.Objecter, count int) ItemStack {
	return ItemStack{stackNew.CallStatic(item, count).Object()}
}

// Count returns the number of items.
func (s ItemStack) Count() int { return int(stackCount.Call(s).Int()) }

// DisplayName returns the stack's name as plain text.
func (s ItemStack) DisplayName() string { return Component{stackHoverName.Call(s).Object()}.Plain() }

// InteractionResult tells the game how an interaction went.
type InteractionResult struct{ jvm.Object }

var (
	resultSuccess = jvm.SF(cResult, "SUCCESS", "Lnet/minecraft/world/InteractionResult$Success;")
	resultConsume = jvm.SF(cResult, "CONSUME", "Lnet/minecraft/world/InteractionResult$Success;")
	resultFail    = jvm.SF(cResult, "FAIL", "Lnet/minecraft/world/InteractionResult$Fail;")
	resultPass    = jvm.SF(cResult, "PASS", "Lnet/minecraft/world/InteractionResult$Pass;")
)

// Success means the interaction happened; further handlers are skipped.
func Success() InteractionResult { return InteractionResult{resultSuccess.GetStatic().Object()} }

// Consume means the interaction happened without the hand swing.
func Consume() InteractionResult { return InteractionResult{resultConsume.GetStatic().Object()} }

// Fail means the interaction is refused; further handlers are skipped.
func Fail() InteractionResult { return InteractionResult{resultFail.GetStatic().Object()} }

// Pass means this handler has no opinion.
func Pass() InteractionResult { return InteractionResult{resultPass.GetStatic().Object()} }
