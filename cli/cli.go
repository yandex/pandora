package cli

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"runtime/pprof"
	"time"

	"github.com/facebookgo/stackerr"
	"github.com/spf13/viper"

	"github.com/yandex/pandora/core/config"
	"github.com/yandex/pandora/core/engine"
	"github.com/yandex/pandora/lib/utils"
)

const Version = "0.2.0"
const defaultConfigFile = "load"

var configSearchDirs = []string{"./", "./config", "/etc/pandora"}

// TODO: make nice spf13/cobra CLI and integrate it with viper
// TODO(skipor): on special command (help or smth else) print list of available plugins

func Run() {
	log.Printf("Pandora v%s\n", Version)
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of Pandora: pandora [<config_filename>]\n"+"<config_filename> is './%s.(yaml|json|...)' by default\n", defaultConfigFile)
		flag.PrintDefaults()
	}
	var (
		example    bool
		monitoring monitoringConfig
	)
	flag.BoolVar(&example, "example", false, "print example config to STDOUT and exit")
	flag.StringVar(&monitoring.CPUProfile, "cpuprofile", "", "write cpu profile to file")
	flag.StringVar(&monitoring.MemProfile, "memprofile", "", "write memory profile to this file")
	flag.BoolVar(&monitoring.Expvar, "expvar", false, "start HTTP server with monitoring variables")
	flag.Parse()

	if example {
		panic("Not implemented yet")
		// TODO: print example config file content
	}

	v := newViper()
	if len(flag.Args()) > 0 {
		v.SetConfigFile(flag.Args()[0])
	}
	err := v.ReadInConfig()
	log.Printf("Reading config from %q", v.ConfigFileUsed())
	if err != nil {
		err = stackerr.Wrap(err)
		log.Fatalf("Config read error: %v", err)
	}
	var conf cliConfig
	err = config.DecodeAndValidate(v.AllSettings(), &conf)
	if err != nil {
		log.Fatal("Config decode error: ", err)
	}

	pandora := engine.New(conf.Engine)
	startMonitoring(monitoring)

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
			log.Fatal("Interrupt timeout exeeded")
		}
	case err = <-promise:
	}
	if err != nil {
		log.Fatal(err)
	}
}

type monitoringConfig struct {
	Expvar     bool   // TODO: struct { Enabled bool; Port string }
	CPUProfile string // TODO: struct { Enabled bool; File string }
	MemProfile string // TODO: struct { Enabled bool; File string }
}

func startMonitoring(conf monitoringConfig) (stop func()) {
	if conf.Expvar {
		go http.ListenAndServe(":1234", nil)
	}
	var stops []func()
	if conf.CPUProfile != "" {
		f, err := os.Create(conf.MemProfile)
		if err != nil {
			log.Fatal(err)
		}
		pprof.StartCPUProfile(f)
		stops = append(stops, func() {
			pprof.StopCPUProfile()
			f.Close()
		})
	}
	if conf.MemProfile != "" {
		f, err := os.Create(conf.MemProfile)
		if err != nil {
			log.Fatal(err)
		}
		stops = append(stops, func() {
			pprof.WriteHeapProfile(f)
			f.Close()
		})
	}
	stop = func() {
		for _, s := range stops {
			s()
		}
	}
	return
}

func newViper() *viper.Viper {
	v := viper.New()
	v.SetConfigName(defaultConfigFile)
	for _, dir := range configSearchDirs {
		v.AddConfigPath(dir)
	}
	return v
}

type cliConfig struct {
	Engine engine.Config `config:",squash"`
	// TODO monitoring config should be there
}
