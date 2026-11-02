# GoCraft SDK

The Go library for writing [GoCraft](https://github.com/GoCraftMC/gocraft) mods.

```go
import (
	"github.com/GoCraftMC/gocraft-sdk/gocraft"
	"github.com/GoCraftMC/gocraft-sdk/registry"
)

func init() {
	gocraft.OnInitialize(func() {
		registry.RegisterItem("ruby", registry.Item{})
	})
}
```

| Package | Purpose |
| --- | --- |
| `gocraft` | Entrypoints, logging, config, hooks |
| `registry` | Items and blocks |
| `event` | Game events |
| `command` | Commands |
| `mc` | Common game types |
| `jvm` | Everything else in the game |

Mods are built with [go.jar](https://github.com/GoCraftMC/gocraft), not `go build`. Building needs cgo and a
C compiler. See the [documentation](https://github.com/GoCraftMC/gocraft-docs).

## License

BSD-4-Clause. This product includes software developed by The GoCraft Authors.
