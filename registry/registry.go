// Package registry adds items and blocks to the game. Call it from gocraft.OnInitialize.
package registry

import (
	"github.com/GoCraftMC/gocraft-sdk/jvm"
	"github.com/GoCraftMC/gocraft-sdk/mc"
)

const (
	cItem      = "net.minecraft.world.item.Item"
	cItemProps = "net.minecraft.world.item.Item$Properties"
	cBlock     = "net.minecraft.world.level.block.Block"
	cBlockProp = "net.minecraft.world.level.block.state.BlockBehaviour$Properties"
	cBlockItem = "net.minecraft.world.item.BlockItem"
	cRegistry  = "net.minecraft.core.Registry"
	cRegs      = "net.minecraft.core.registries.Registries"
	cBuiltin   = "net.minecraft.core.registries.BuiltInRegistries"
	cKey       = "net.minecraft.resources.ResourceKey"
	cTabs      = "net.minecraft.world.item.CreativeModeTabs"
	cTabEvents = "net.fabricmc.fabric.api.creativetab.v1.CreativeModeTabEvents"
	cTabOutput = "net.minecraft.world.item.CreativeModeTab$Output"
	cEvent     = "net.fabricmc.fabric.api.event.Event"

	descItemProps  = "Lnet/minecraft/world/item/Item$Properties;"
	descBlockProps = "Lnet/minecraft/world/level/block/state/BlockBehaviour$Properties;"
	descUseItem    = "use(Lnet/minecraft/world/level/Level;Lnet/minecraft/world/entity/player/Player;Lnet/minecraft/world/InteractionHand;)Lnet/minecraft/world/InteractionResult;"
	descUseBlock   = "useWithoutItem(Lnet/minecraft/world/level/block/state/BlockState;Lnet/minecraft/world/level/Level;Lnet/minecraft/core/BlockPos;Lnet/minecraft/world/entity/player/Player;Lnet/minecraft/world/phys/BlockHitResult;)Lnet/minecraft/world/InteractionResult;"
	descStepOn     = "stepOn(Lnet/minecraft/world/level/Level;Lnet/minecraft/core/BlockPos;Lnet/minecraft/world/level/block/state/BlockState;Lnet/minecraft/world/entity/Entity;)V"
)

var (
	keyCreate     = jvm.SM(cKey, "create", "(Lnet/minecraft/resources/ResourceKey;Lnet/minecraft/resources/Identifier;)Lnet/minecraft/resources/ResourceKey;")
	regItemKey    = jvm.SF(cRegs, "ITEM", "Lnet/minecraft/resources/ResourceKey;")
	regBlockKey   = jvm.SF(cRegs, "BLOCK", "Lnet/minecraft/resources/ResourceKey;")
	builtinItem   = jvm.SF(cBuiltin, "ITEM", "Lnet/minecraft/core/DefaultedRegistry;")
	builtinBlock  = jvm.SF(cBuiltin, "BLOCK", "Lnet/minecraft/core/DefaultedRegistry;")
	registerByKey = jvm.SM(cRegistry, "register", "(Lnet/minecraft/core/Registry;Lnet/minecraft/resources/ResourceKey;Ljava/lang/Object;)Ljava/lang/Object;")

	itemProps          = jvm.Ctor(cItemProps, "()V")
	itemPropsSetID     = jvm.M(cItemProps, "setId", "(Lnet/minecraft/resources/ResourceKey;)"+descItemProps)
	itemPropsStacksTo  = jvm.M(cItemProps, "stacksTo", "(I)"+descItemProps)
	itemPropsFireproof = jvm.M(cItemProps, "fireResistant", "()"+descItemProps)
	itemPropsBlockDesc = jvm.M(cItemProps, "useBlockDescriptionPrefix", "()"+descItemProps)
	itemNew            = jvm.Ctor(cItem, "("+descItemProps+")V")
	blockItemNew       = jvm.Ctor(cBlockItem, "(Lnet/minecraft/world/level/block/Block;"+descItemProps+")V")

	blockPropsOf       = jvm.SM(cBlockProp, "of", "()"+descBlockProps)
	blockPropsSetID    = jvm.M(cBlockProp, "setId", "(Lnet/minecraft/resources/ResourceKey;)"+descBlockProps)
	blockPropsStrength = jvm.M(cBlockProp, "strength", "(FF)"+descBlockProps)
	blockPropsLight    = jvm.M(cBlockProp, "lightLevel", "(Ljava/util/function/ToIntFunction;)"+descBlockProps)
	blockPropsTool     = jvm.M(cBlockProp, "requiresCorrectToolForDrops", "()"+descBlockProps)
	blockPropsInstant  = jvm.M(cBlockProp, "instabreak", "()"+descBlockProps)
	blockNew           = jvm.Ctor(cBlock, "("+descBlockProps+")V")

	tabEvent    = jvm.SM(cTabEvents, "modifyOutputEvent", "(Lnet/minecraft/resources/ResourceKey;)Lnet/fabricmc/fabric/api/event/Event;")
	eventListen = jvm.M(cEvent, "register", "(Ljava/lang/Object;)V")
	tabAccept   = jvm.M(cTabOutput, "accept", "(Lnet/minecraft/world/level/ItemLike;)V")
)

func key(registry *jvm.LazyField, id mc.Identifier) jvm.Object {
	return keyCreate.CallStatic(registry.GetStatic(), id).Object()
}

// Item describes a new item.
type Item struct {
	// MaxStack is the stack size; 0 means the default of 64.
	MaxStack int
	// FireResistant items survive fire and lava.
	FireResistant bool
	// OnUse runs when a player right-clicks with the item.
	OnUse func(ctx UseItem) mc.InteractionResult
}

// UseItem is the context for Item.OnUse.
type UseItem struct {
	Level  mc.Level
	Player mc.Player
	Hand   jvm.Object
}

// RegisterItem adds an item. A bare id uses this mod's namespace.
func RegisterItem(id string, def Item) mc.Item {
	identifier := mc.ID(id)
	itemKey := key(regItemKey, identifier)

	props := itemPropsSetID.Call(itemProps.CallStatic(), itemKey).Object()
	if def.MaxStack > 0 {
		props = itemPropsStacksTo.Call(props, def.MaxStack).Object()
	}
	if def.FireResistant {
		props = itemPropsFireproof.Call(props).Object()
	}

	var item jvm.Object
	if def.OnUse == nil {
		item = itemNew.CallStatic(props).Object()
	} else {
		class := jvm.Extend(jvm.FindClass(cItem), id, jvm.Impl{
			descUseItem: func(a *jvm.Args) any {
				return def.OnUse(UseItem{
					Level:  mc.Level{Object: a.Object(1)},
					Player: mc.Player{Object: a.Object(2)},
					Hand:   a.Object(3),
				})
			},
		})
		item = class.New("("+descItemProps+")V", props)
	}

	registerByKey.CallStatic(builtinItem.GetStatic(), itemKey, item)
	return mc.Item{Object: item}
}

// Block describes a new block.
type Block struct {
	// Hardness is how long the block takes to mine; stone is 1.5.
	Hardness float32
	// Resistance is how well it withstands explosions; stone is 6.
	Resistance float32
	// Light is the light level emitted, 0 to 15.
	Light int
	// RequiresTool means it only drops when mined with the correct tool.
	RequiresTool bool
	// Instant means it breaks in one hit.
	Instant bool
	// NoItem skips creating the matching BlockItem.
	NoItem bool
	// OnUse runs when a player right-clicks the block with an empty hand.
	OnUse func(ctx UseBlock) mc.InteractionResult
	// OnStep runs when an entity walks on the block.
	OnStep func(ctx StepOn)
}

// UseBlock is the context for Block.OnUse.
type UseBlock struct {
	State  mc.BlockState
	Level  mc.Level
	Pos    mc.BlockPos
	Player mc.Player
}

// StepOn is the context for Block.OnStep.
type StepOn struct {
	Level  mc.Level
	Pos    mc.BlockPos
	State  mc.BlockState
	Entity mc.Entity
}

// RegisteredBlock holds a block and, unless disabled, its item.
type RegisteredBlock struct {
	mc.Block
	Item mc.Item
}

// RegisterBlock adds a block and its item. A bare id uses this mod's namespace.
func RegisterBlock(id string, def Block) RegisteredBlock {
	identifier := mc.ID(id)
	blockKey := key(regBlockKey, identifier)

	props := blockPropsOf.CallStatic().Object()
	props = blockPropsSetID.Call(props, blockKey).Object()
	if def.Instant {
		props = blockPropsInstant.Call(props).Object()
	} else {
		props = blockPropsStrength.Call(props, def.Hardness, def.Resistance).Object()
	}
	if def.RequiresTool {
		props = blockPropsTool.Call(props).Object()
	}
	if def.Light > 0 {
		light := def.Light
		fn := jvm.Func("java.util.function.ToIntFunction", "applyAsInt(Ljava/lang/Object;)I", func(*jvm.Args) any { return light })
		props = blockPropsLight.Call(props, fn).Object()
	}

	methods := jvm.Impl{}
	if def.OnUse != nil {
		methods[descUseBlock] = func(a *jvm.Args) any {
			return def.OnUse(UseBlock{
				State:  mc.BlockState{Object: a.Object(1)},
				Level:  mc.Level{Object: a.Object(2)},
				Pos:    mc.BlockPos{Object: a.Object(3)},
				Player: mc.Player{Object: a.Object(4)},
			})
		}
	}
	if def.OnStep != nil {
		methods[descStepOn] = func(a *jvm.Args) any {
			def.OnStep(StepOn{
				Level:  mc.Level{Object: a.Object(1)},
				Pos:    mc.BlockPos{Object: a.Object(2)},
				State:  mc.BlockState{Object: a.Object(3)},
				Entity: mc.Entity{Object: a.Object(4)},
			})
			return nil
		}
	}

	var block jvm.Object
	if len(methods) == 0 {
		block = blockNew.CallStatic(props).Object()
	} else {
		block = jvm.Extend(jvm.FindClass(cBlock), id, methods).New("("+descBlockProps+")V", props)
	}
	registerByKey.CallStatic(builtinBlock.GetStatic(), blockKey, block)

	result := RegisteredBlock{Block: mc.Block{Object: block}}
	if !def.NoItem {
		itemKey := key(regItemKey, identifier)
		itemProp := itemPropsBlockDesc.Call(itemPropsSetID.Call(itemProps.CallStatic(), itemKey)).Object()
		item := blockItemNew.CallStatic(block, itemProp).Object()
		registerByKey.CallStatic(builtinItem.GetStatic(), itemKey, item)
		result.Item = mc.Item{Object: item}
	}
	return result
}

// Tab names one of the vanilla creative tabs.
type Tab string

// Vanilla creative tabs.
const (
	TabBuildingBlocks Tab = "BUILDING_BLOCKS"
	TabColoredBlocks  Tab = "COLORED_BLOCKS"
	TabNaturalBlocks  Tab = "NATURAL_BLOCKS"
	TabFunctional     Tab = "FUNCTIONAL_BLOCKS"
	TabRedstone       Tab = "REDSTONE_BLOCKS"
	TabTools          Tab = "TOOLS_AND_UTILITIES"
	TabCombat         Tab = "COMBAT"
	TabFood           Tab = "FOOD_AND_DRINKS"
	TabIngredients    Tab = "INGREDIENTS"
	TabSpawnEggs      Tab = "SPAWN_EGGS"
)

// AddToTab lists items in a creative tab, after what is already there.
func AddToTab(tab Tab, items ...jvm.Objecter) {
	tabKey := jvm.FindClass(cTabs).StaticField(string(tab), "Lnet/minecraft/resources/ResourceKey;").GetStatic()
	listener := jvm.Func(cTabEvents+"$ModifyOutput", "modifyOutput(Lnet/fabricmc/fabric/api/creativetab/v1/FabricCreativeModeTabOutput;)V", func(a *jvm.Args) any {
		output := a.Object(0)
		for _, item := range items {
			tabAccept.Call(output, item)
		}
		return nil
	})
	eventListen.Call(tabEvent.CallStatic(tabKey), listener)
}
