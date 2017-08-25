package cli

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"runtime/pprof"
	"syscall"
	"time"

	"github.com/spf13/viper"
	"go.uber.org/zap"

	"github.com/yandex/pandora/core/config"
	"github.com/yandex/pandora/core/engine"
)

const Version = "0.2.0"
const defaultConfigFile = "load"

var configSearchDirs = []string{"./", "./config", "/etc/pandora"}

type cliConfig struct {
	Engine engine.Config `config:",squash"`
	// TODO(skipor): monitoring, logging configs should be there
}

// TODO(skipor): make nice spf13/cobra CLI and integrate it with viper
// TODO(skipor): on special command (help or smth else) print list of available plugins

func Run() {
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

	log, conf := readConfig()
	closeMonitoring := startMonitoring(monitoring)
	defer closeMonitoring()
	m := newEngineMetrics()
	startReport(m)

	pandora := engine.New(log, m, conf.Engine)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go handleSignals(log, cancel)

	err := pandora.Run(ctx)
	if err != nil {
		const awaitTimeout = 3 * time.Second
		log.Error("Engine run failed. Awaiting started tasks.", zap.Error(err), zap.Duration("timeout", awaitTimeout))
		time.AfterFunc(awaitTimeout, func() {
			log.Fatal("Engine tasks timeout exceeded.")
		})
		pandora.Wait()
		os.Exit(1)
	}
	log.Info("Engine run successfully finished")
}

func readConfig() (*zap.Logger, cliConfig) {
	log, err := zap.NewDevelopment(zap.AddCaller())
	if err != nil {
		panic(err)
	}
	log.Info("Pandora started", zap.String("version", Version))
	zap.ReplaceGlobals(log)
	zap.RedirectStdLog(log)

	v := newViper()
	if len(flag.Args()) > 0 {
		v.SetConfigFile(flag.Args()[0])
	}
	err = v.ReadInConfig()
	log.Info("Reading config", zap.String("file", v.ConfigFileUsed()))
	if err != nil {
		log.Fatal("Config read failed", zap.Error(err))
	}
	var conf cliConfig
	err = config.DecodeAndValidate(v.AllSettings(), &conf)
	if err != nil {
		log.Fatal("Config decode failed", zap.Error(err))
	}
	return log, conf
}

func newViper() *viper.Viper {
	v := viper.New()
	v.SetConfigName(defaultConfigFile)
	for _, dir := range configSearchDirs {
		v.AddConfigPath(dir)
	}
	return v
}

func handleSignals(log *zap.Logger, interrupt func()) {
	sigs := make(chan os.Signal, 2)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	select {
	case sig := <-sigs:
		switch sig {
		case syscall.SIGINT:
			const interruptTimeout = 5 * time.Second
			log.Info("SIGINT received. Trying to stop gracefully.", zap.Duration("timeout", interruptTimeout))
			interrupt()
			select {
			case <-time.After(interruptTimeout):
				log.Fatal("Interrupt timeout exceeded")
			case sig := <-sigs:
				log.Fatal("Another signal received. Quiting.", zap.Stringer("signal", sig))
			}
		case syscall.SIGTERM:
			log.Info("SIGTERM received. Quiting.")
		default:
			log.Info("Unexpected signal received. Quiting.", zap.Stringer("signal", sig))
		}
	}
}

type monitoringConfig struct {
	Expvar     bool   // TODO: struct { Enabled bool; Port string }
	CPUProfile string // TODO: struct { Enabled bool; File string }
	MemProfile string // TODO: struct { Enabled bool; File string }
}

func startMonitoring(conf monitoringConfig) (stop func()) {
	if conf.Expvar {
		go func() {
			err := http.ListenAndServe(":1234", nil)
			zap.L().Fatal("Monitoring server failed", zap.Error(err))
		}()
	}
	var stops []func()
	if conf.CPUProfile != "" {
		f, err := os.Create(conf.MemProfile)
		if err != nil {
			zap.L().Fatal("Memory profile file create fail", zap.Error(err))
		}
		pprof.StartCPUProfile(f)
		stops = append(stops, func() {
			pprof.StopCPUProfile()
			f.Close()
		})
	}
	if conf.MemProfile != "" {
		f, err := os.Create(conf.CPUProfile)
		if err != nil {
			zap.L().Fatal("CPU profile file create fail", zap.Error(err))
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
