// Package command declares Brigadier commands as a tree of nodes:
//
//	command.Register(
//		command.Literal("greet").Then(
//			command.Arg("name", command.Word()).Executes(func(c *command.Context) int {
//				c.Reply("Hello, " + c.String("name"))
//				return 1
//			}),
//		),
//	)
package command

import (
	"github.com/GoCraftMC/gocraft-sdk/jvm"
	"github.com/GoCraftMC/gocraft-sdk/mc"
)

const (
	cCommands = "net.minecraft.commands.Commands"
	cSource   = "net.minecraft.commands.CommandSourceStack"
	cBuilder  = "com.mojang.brigadier.builder.ArgumentBuilder"
	cCallback = "net.fabricmc.fabric.api.command.v2.CommandRegistrationCallback"
	cStrArg   = "com.mojang.brigadier.arguments.StringArgumentType"
	cIntArg   = "com.mojang.brigadier.arguments.IntegerArgumentType"
	cBoolArg  = "com.mojang.brigadier.arguments.BoolArgumentType"
	cDblArg   = "com.mojang.brigadier.arguments.DoubleArgumentType"

	descBuilder = "Lcom/mojang/brigadier/builder/ArgumentBuilder;"
	descContext = "Lcom/mojang/brigadier/context/CommandContext;"
)

var (
	literal  = jvm.SM(cCommands, "literal", "(Ljava/lang/String;)Lcom/mojang/brigadier/builder/LiteralArgumentBuilder;")
	argument = jvm.SM(cCommands, "argument", "(Ljava/lang/String;Lcom/mojang/brigadier/arguments/ArgumentType;)Lcom/mojang/brigadier/builder/RequiredArgumentBuilder;")
	then     = jvm.M(cBuilder, "then", "("+descBuilder+")"+descBuilder)
	executes = jvm.M(cBuilder, "executes", "(Lcom/mojang/brigadier/Command;)"+descBuilder)
	requires = jvm.M(cBuilder, "requires", "(Ljava/util/function/Predicate;)"+descBuilder)
	register = jvm.M("com.mojang.brigadier.CommandDispatcher", "register", "(Lcom/mojang/brigadier/builder/LiteralArgumentBuilder;)Lcom/mojang/brigadier/tree/LiteralCommandNode;")
	eventAdd = jvm.M("net.fabricmc.fabric.api.event.Event", "register", "(Ljava/lang/Object;)V")

	getSource = jvm.M("com.mojang.brigadier.context.CommandContext", "getSource", "()Ljava/lang/Object;")
	getString = jvm.SM(cStrArg, "getString", "("+descContext+"Ljava/lang/String;)Ljava/lang/String;")
	getInt    = jvm.SM(cIntArg, "getInteger", "("+descContext+"Ljava/lang/String;)I")
	getBool   = jvm.SM(cBoolArg, "getBool", "("+descContext+"Ljava/lang/String;)Z")
	getDouble = jvm.SM(cDblArg, "getDouble", "("+descContext+"Ljava/lang/String;)D")

	sendSuccess = jvm.M(cSource, "sendSuccess", "(Ljava/util/function/Supplier;Z)V")
	sendFailure = jvm.M(cSource, "sendFailure", "(Lnet/minecraft/network/chat/Component;)V")
	getPlayer   = jvm.M(cSource, "getPlayer", "()Lnet/minecraft/server/level/ServerPlayer;")
	getServer   = jvm.M(cSource, "getServer", "()Lnet/minecraft/server/MinecraftServer;")
	getLevel    = jvm.M(cSource, "getLevel", "()Lnet/minecraft/server/level/ServerLevel;")
	getName     = jvm.M(cSource, "getTextName", "()Ljava/lang/String;")
)

// ArgType is a Brigadier argument parser.
type ArgType func() jvm.Object

func simple(class, method, ret string) ArgType {
	m := jvm.SM(class, method, "()L"+ret+";")
	return func() jvm.Object { return m.CallStatic().Object() }
}

// Word parses a single word.
func Word() ArgType {
	return simple(cStrArg, "word", "com/mojang/brigadier/arguments/StringArgumentType")
}

// QuotedString parses a word or a "quoted string".
func QuotedString() ArgType {
	return simple(cStrArg, "string", "com/mojang/brigadier/arguments/StringArgumentType")
}

// GreedyString parses all remaining input.
func GreedyString() ArgType {
	return simple(cStrArg, "greedyString", "com/mojang/brigadier/arguments/StringArgumentType")
}

// Bool parses true or false.
func Bool() ArgType {
	return simple(cBoolArg, "bool", "com/mojang/brigadier/arguments/BoolArgumentType")
}

// Int parses an integer.
func Int() ArgType {
	return simple(cIntArg, "integer", "com/mojang/brigadier/arguments/IntegerArgumentType")
}

// IntRange parses an integer between min and max.
func IntRange(min, max int) ArgType {
	m := jvm.SM(cIntArg, "integer", "(II)Lcom/mojang/brigadier/arguments/IntegerArgumentType;")
	return func() jvm.Object { return m.CallStatic(min, max).Object() }
}

// Double parses a floating point number.
func Double() ArgType {
	return simple(cDblArg, "doubleArg", "com/mojang/brigadier/arguments/DoubleArgumentType")
}

// Context is one execution of a command.
type Context struct {
	obj    jvm.Object
	source jvm.Object
}

// Source returns the raw CommandSourceStack.
func (c *Context) Source() jvm.Object { return c.source }

// Reply sends a success message to whoever ran the command.
func (c *Context) Reply(message string) {
	text := mc.Text(message)
	sendSuccess.Call(c.source, jvm.Supplier(func() any { return text }), false)
}

// Fail sends a red failure message to whoever ran the command.
func (c *Context) Fail(message string) { sendFailure.Call(c.source, mc.Text(message)) }

// Player returns the player who ran the command; ok is false for the console or command blocks.
func (c *Context) Player() (player mc.Player, ok bool) {
	p := getPlayer.Call(c.source).Object()
	return mc.Player{Object: p}, !p.IsNull()
}

// Server returns the server.
func (c *Context) Server() mc.Server { return mc.Server{Object: getServer.Call(c.source).Object()} }

// Level returns the world the command ran in.
func (c *Context) Level() mc.Level { return mc.Level{Object: getLevel.Call(c.source).Object()} }

// SenderName returns the name of whoever ran the command.
func (c *Context) SenderName() string { return getName.Call(c.source).String() }

// String returns a string argument.
func (c *Context) String(name string) string { return getString.CallStatic(c.obj, name).String() }

// Int returns an integer argument.
func (c *Context) Int(name string) int { return int(getInt.CallStatic(c.obj, name).Int()) }

// Bool returns a boolean argument.
func (c *Context) Bool(name string) bool { return getBool.CallStatic(c.obj, name).Bool() }

// Double returns a floating point argument.
func (c *Context) Double(name string) float64 { return getDouble.CallStatic(c.obj, name).Double() }

// Node is one part of a command tree.
type Node struct {
	create   func() jvm.Object
	children []*Node
	exec     func(*Context) int
	check    func(*Context) bool
}

// Literal matches an exact word.
func Literal(name string) *Node {
	return &Node{create: func() jvm.Object { return literal.CallStatic(name).Object() }}
}

// Arg parses a named argument.
func Arg(name string, kind ArgType) *Node {
	return &Node{create: func() jvm.Object { return argument.CallStatic(name, kind()).Object() }}
}

// Then adds child nodes.
func (n *Node) Then(children ...*Node) *Node {
	n.children = append(n.children, children...)
	return n
}

// Executes sets what runs when the command stops at this node. Return 1 for success, or a count.
func (n *Node) Executes(fn func(*Context) int) *Node {
	n.exec = fn
	return n
}

// Requires hides this node from sources that fn rejects. The context has no arguments yet, so use
// only its source, player, server and level.
func (n *Node) Requires(fn func(*Context) bool) *Node {
	n.check = fn
	return n
}

func (n *Node) build() jvm.Object {
	builder := n.create()

	if n.exec != nil {
		exec := n.exec
		cmd := jvm.Func("com.mojang.brigadier.Command", "run("+descContext+")I", func(a *jvm.Args) any {
			ctx := a.Object(0)
			return exec(&Context{obj: ctx, source: getSource.Call(ctx).Object()})
		})
		executes.Call(builder, cmd)
	}

	if n.check != nil {
		check := n.check
		predicate := jvm.Func("java.util.function.Predicate", "test(Ljava/lang/Object;)Z", func(a *jvm.Args) any {
			return check(&Context{source: a.Object(0)})
		})
		requires.Call(builder, predicate)
	}

	for _, child := range n.children {
		then.Call(builder, child.build())
	}
	return builder
}

// Register adds top-level commands. Each node must be a Literal. Call it from gocraft.OnInitialize.
func Register(nodes ...*Node) {
	event := jvm.FindClass(cCallback).StaticField("EVENT", "Lnet/fabricmc/fabric/api/event/Event;").GetStatic()
	listener := jvm.Func(cCallback,
		"register(Lcom/mojang/brigadier/CommandDispatcher;Lnet/minecraft/commands/CommandBuildContext;Lnet/minecraft/commands/Commands$CommandSelection;)V",
		func(a *jvm.Args) any {
			dispatcher := a.Object(0)
			for _, node := range nodes {
				register.Call(dispatcher, node.build())
			}
			return nil
		})
	eventAdd.Call(event, listener)
}
