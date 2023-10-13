[Home](../index.md)

---

# Custom guns

- [Basic tutorial](#basic-tutorial)
- [gRPC](#grpc)
- [Websockets](#websockets)

## Basic tutorial

You can create you own Golang-based gun with _pandora_.

There is an example of custom gun shooting via gRPC.

We create a new gun and define `shoot` method for it w/ our test logic.

You can find examples of custom guns' code below:
-  [gRPC](#gRPC)
-  [Websockets](#Websockets)

Now it's time to compile our gun. Install deps and compile your custom gun file (`go build my_custom_gun.go`).

After that step you'll get `my_custom_gun` binary file, it is compiled pandora with your custom gun inside.

Now its time to create `load.yaml`:

```yaml
pools:
  - id: HTTP pool
    gun:
      type: my_custom_gun_name   # custom gun name (specified at `register.Gun("my_custom_gun_name", ...`)
      target: "your_grpc_host:your_grpc_port"
    ammo:
      type: custom_provider
      source:
        type: file
        path: ./json.ammo
    result:
      type: phout
      destination: ./phout.log
    rps: { duration: 30s, type: line,  from: 1, to: 2 }
    startup:
      type: once
      times: 10
    log:
```

And create ammofile `./json.ammo`:

```yaml
{"tag": "/MyCase1", "Param1": "146837693,146837692,146837691"}
{"tag": "/MyCase2", "Param2": "555", "Param1": "500002"}
```


We are ready to shoot. Try it.


## gRPC

```go
// create a package
package main

// import some pandora stuff
// and stuff you need for your scenario
// and protobuf contracts for your grpc service

import (
        "log"
        "context"
        "strconv"
        "strings"
        "time"

        "github.com/golang/protobuf/ptypes/timestamp"
        "github.com/satori/go.uuid"
        "github.com/spf13/afero"
        "google.golang.org/grpc"
        pb "my_package/my_protobuf_contracts"

        "github.com/yandex/pandora/cli"
        "github.com/yandex/pandora/components/phttp/import"
        "github.com/yandex/pandora/core"
        "github.com/yandex/pandora/core/aggregator/netsample"
        "github.com/yandex/pandora/core/import"
        "github.com/yandex/pandora/core/register"
)

type Ammo struct {
        Tag         string
        Param1      string
        Param2      string
        Param3      string
}

type Sample struct {
        URL              string
        ShootTimeSeconds float64
}

type GunConfig struct {
        Target string `validate:"required"` // Configuration will fail, without target defined
}

type Gun struct {
        // Configured on construction.
        client grpc.ClientConn
        conf   GunConfig
        // Configured on Bind, before shooting
        aggr core.Aggregator // May be your custom Aggregator.
        core.GunDeps
}

func NewGun(conf GunConfig) *Gun {
        return &Gun{conf: conf}
}

func (g *Gun) Bind(aggr core.Aggregator, deps core.GunDeps) error {
        // create gRPC stub at gun initialization
        conn, err := grpc.Dial(
                g.conf.Target,
                grpc.WithInsecure(),
                grpc.WithTimeout(time.Second),
                grpc.WithUserAgent("load test, pandora custom shooter"))
        if err != nil {
                log.Fatalf("FATAL: %s", err)
        }
        g.client = *conn
        g.aggr = aggr
        g.GunDeps = deps
        return nil
}

func (g *Gun) Shoot(ammo core.Ammo) {
        customAmmo := ammo.(*Ammo)
        g.shoot(customAmmo)
}


func (g *Gun) case1_method(client pb.MyClient, ammo *Ammo) int {
        code := 0
        // prepare list of ids from ammo
        var itemIDs []int64
        for _, id := range strings.Split(ammo.Param1, ",") {
                if id == "" {
                        continue
                }
                itemID, err := strconv.ParseInt(id, 10, 64)
                if err != nil {
                        log.Printf("Ammo parse FATAL: %s", err)
                        code = 314
                }
                itemIDs = append(itemIDs, itemID)
        }

        out, err := client.GetSomeData(
                context.TODO(), &pb.ItemsRequest{
                        itemIDs})

        if err != nil {
                log.Printf("FATAL: %s", err)
                code = 500
        }

        if out != nil {
                code = 200
        }
        return code
}

func (g *Gun) case2_method(client pb.MyClient, ammo *Ammo) int {
        code := 0
        // prepare item_id and warehouse_id
        item_id, err := strconv.ParseInt(ammo.Param1, 10, 0)
        if err != nil {
                log.Printf("Failed to parse ammo FATAL", err)
                code = 314
        }
        warehouse_id, err2 := strconv.ParseInt(ammo.Param2, 10, 0)
        if err2 != nil {
                log.Printf("Failed to parse ammo FATAL", err2)
                code = 314
        }

        items := []*pb.SomeItem{}
        items = append(items, &pb.SomeItem{
                item_id,
                warehouse_id,
                1,
                &timestamp.Timestamp{time.Now().Unix(), 111}
        })

        out2, err3 := client.GetSomeDataSecond(
                context.TODO(), &pb.SomeRequest{
                        uuid.Must(uuid.NewV4()).String(),
                        1,
                        items})
        if err3 != nil {
                log.Printf("FATAL", err3)
                code = 316
        }

        if out2 != nil {
                code = 200
        }


        return code
}

func (g *Gun) shoot(ammo *Ammo) {
        code := 0
        sample := netsample.Acquire(ammo.Tag)

        conn := g.client
        client := pb.NewClient(&conn)

        switch ammo.Tag {
        case "/MyCase1":
            code = g.case1_method(client, ammo)
        case "/MyCase2":
            code = g.case2_method(client, ammo)
        default:
            code = 404
        }

        defer func() {
                sample.SetProtoCode(code)
                g.aggr.Report(sample)
        }()
}

func main() {
        //debug.SetGCPercent(-1)
        // Standard imports.
        fs := afero.NewOsFs()
        coreimport.Import(fs)
        // May not be imported, if you don't need http guns and etc.
        phttp.Import(fs)

        // Custom imports. Integrate your custom types into configuration system.
        coreimport.RegisterCustomJSONProvider("custom_provider", func() core.Ammo { return &Ammo{} })

        register.Gun("my_custom_gun_name", NewGun, func() GunConfig {
                return GunConfig{
                        Target: "default target",
                }
        })

        cli.Run()
}
```

## Websockets

```go
// create a package
package main

// import some pandora stuff
// and stuff you need for your scenario
// and protobuf contracts for your grpc service

import (
        "log"
        "context"
        "strconv"
        "strings"
        "time"

        "github.com/golang/protobuf/ptypes/timestamp"
        "github.com/satori/go.uuid"
        "github.com/spf13/afero"
        "google.golang.org/grpc"
        pb "my_package/my_protobuf_contracts"

        "github.com/yandex/pandora/cli"
        "github.com/yandex/pandora/components/phttp/import"
        "github.com/yandex/pandora/core"
        "github.com/yandex/pandora/core/aggregator/netsample"
        "github.com/yandex/pandora/core/import"
        "github.com/yandex/pandora/core/register"
)

type Ammo struct {
        Tag         string
        Param1      string
        Param2      string
        Param3      string
}

type Sample struct {
        URL              string
        ShootTimeSeconds float64
}

type GunConfig struct {
        Target string `validate:"required"` // Configuration will fail, without target defined
}

type Gun struct {
        // Configured on construction.
        client grpc.ClientConn
        conf   GunConfig
        // Configured on Bind, before shooting
        aggr core.Aggregator // May be your custom Aggregator.
        core.GunDeps
}

func NewGun(conf GunConfig) *Gun {
        return &Gun{conf: conf}
}

func (g *Gun) Bind(aggr core.Aggregator, deps core.GunDeps) error {
        // create gRPC stub at gun initialization
        conn, err := grpc.Dial(
                g.conf.Target,
                grpc.WithInsecure(),
                grpc.WithTimeout(time.Second),
                grpc.WithUserAgent("load test, pandora custom shooter"))
        if err != nil {
                log.Fatalf("FATAL: %s", err)
        }
        g.client = *conn
        g.aggr = aggr
        g.GunDeps = deps
        return nil
}

func (g *Gun) Shoot(ammo core.Ammo) {
        customAmmo := ammo.(*Ammo)
        g.shoot(customAmmo)
}


func (g *Gun) case1_method(client pb.MyClient, ammo *Ammo) int {
        code := 0
        // prepare list of ids from ammo
        var itemIDs []int64
        for _, id := range strings.Split(ammo.Param1, ",") {
                if id == "" {
                        continue
                }
                itemID, err := strconv.ParseInt(id, 10, 64)
                if err != nil {
                        log.Printf("Ammo parse FATAL: %s", err)
                        code = 314
                }
                itemIDs = append(itemIDs, itemID)
        }

        out, err := client.GetSomeData(
                context.TODO(), &pb.ItemsRequest{
                        itemIDs})

        if err != nil {
                log.Printf("FATAL: %s", err)
                code = 500
        }

        if out != nil {
                code = 200
        }
        return code
}

func (g *Gun) case2_method(client pb.MyClient, ammo *Ammo) int {
        code := 0
        // prepare item_id and warehouse_id
        item_id, err := strconv.ParseInt(ammo.Param1, 10, 0)
        if err != nil {
                log.Printf("Failed to parse ammo FATAL", err)
                code = 314
        }
        warehouse_id, err2 := strconv.ParseInt(ammo.Param2, 10, 0)
        if err2 != nil {
                log.Printf("Failed to parse ammo FATAL", err2)
                code = 314
        }

        items := []*pb.SomeItem{}
        items = append(items, &pb.SomeItem{
                item_id,
                warehouse_id,
                1,
                &timestamp.Timestamp{time.Now().Unix(), 111}
        })

        out2, err3 := client.GetSomeDataSecond(
                context.TODO(), &pb.SomeRequest{
                        uuid.Must(uuid.NewV4()).String(),
                        1,
                        items})
        if err3 != nil {
                log.Printf("FATAL", err3)
                code = 316
        }

        if out2 != nil {
                code = 200
        }


        return code
}

func (g *Gun) shoot(ammo *Ammo) {
        code := 0
        sample := netsample.Acquire(ammo.Tag)

        conn := g.client
        client := pb.NewClient(&conn)

        switch ammo.Tag {
        case "/MyCase1":
            code = g.case1_method(client, ammo)
        case "/MyCase2":
            code = g.case2_method(client, ammo)
        default:
            code = 404
        }

        defer func() {
                sample.SetProtoCode(code)
                g.aggr.Report(sample)
        }()
}

func main() {
        //debug.SetGCPercent(-1)
        // Standard imports.
        fs := afero.NewOsFs()
        coreimport.Import(fs)
        // May not be imported, if you don't need http guns and etc.
        phttp.Import(fs)

        // Custom imports. Integrate your custom types into configuration system.
        coreimport.RegisterCustomJSONProvider("custom_provider", func() core.Ammo { return &Ammo{} })

        register.Gun("my_custom_gun_name", NewGun, func() GunConfig {
                return GunConfig{
                        Target: "default target",
                }
        })

        cli.Run()
}
```

---

[Home](../index.md)