package cmd

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"runtime/pprof"
	"time"

	"golang.org/x/net/context"

	"github.com/yandex/pandora/aggregate"
	"github.com/yandex/pandora/ammo"
	"github.com/yandex/pandora/config"
	"github.com/yandex/pandora/engine"
	"github.com/yandex/pandora/extpoints"
	"github.com/yandex/pandora/limiter"
	"github.com/yandex/pandora/utils"
)

func init() {
	// inject ammo providers
	extpoints.AmmoProviders.Register(ammo.NewHttpProvider, "jsonline/http")
	extpoints.AmmoProviders.Register(ammo.NewHttpProvider, "jsonline/spdy")
	extpoints.AmmoProviders.Register(ammo.NewLogAmmoProvider, "dummy/log")

	// inject result listeners
	extpoints.ResultListeners.Register(aggregate.NewLoggingResultListener, "log/simple")
	extpoints.ResultListeners.Register(aggregate.GetPhoutResultListener, "log/phout")

	// inject limiters
	extpoints.Limiters.Register(limiter.NewPeriodicFromConfig, "periodic")
	extpoints.Limiters.Register(limiter.NewCompositeFromConfig, "composite")
	extpoints.Limiters.Register(limiter.NewUnlimitedFromConfig, "unlimited")
}

func Run() {
	fmt.Printf("Pandora v%s\n", Version)
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of Pandora: pandora [<config_filename>]\n"+
			"<config_filename> is './load.json' by default\n")
		flag.PrintDefaults()
	}
	example := flag.Bool("example", false, "print example config to STDOUT and exit")
	cpuprofile := flag.String("cpuprofile", "", "write cpu profile to file")
	flag.Parse()

	if *example {
		fmt.Printf(exampleConfig)
		return
	}

	configFileName := "./load.json"
	if len(flag.Args()) > 0 {
		configFileName = flag.Args()[0]
	}
	log.Printf("Reading config from '%s'...\n", configFileName)
	jsonDoc, err := ioutil.ReadFile(configFileName)
	if err != nil {
		log.Printf("Could not read config from file: %s", err)
		return
	}
	cfg, err := config.NewGlobalFromJSON(jsonDoc)
	if err != nil {
		log.Printf("Could not unmarshal config from json: %s", err)
		return
	}

	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			log.Fatal(err)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	pandora := engine.New(cfg)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	promise := utils.Promise(func() error { return pandora.Serve(ctx) })

	select {
	case <-utils.NotifyInterrupt():
		log.Print("Interrupting by signal, trying to stop")
		cancel()
		select {
		case err = <-promise:
		case <-time.After(time.Second * 5):
			err = fmt.Errorf("timeout exceeded")
		}
	case err = <-promise:
	}
	if err != nil {
		log.Fatal(err)
	}
}
