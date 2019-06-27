package cli

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"runtime/pprof"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/viper"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/yandex/pandora/core/config"
	"github.com/yandex/pandora/core/engine"
	"github.com/yandex/pandora/lib/zaputil"
)

const Version = "0.3.0"
const defaultConfigFile = "load"
const stdinConfigSelector = "-"

var useStdinConfig = false
var configSearchDirs = []string{"./", "./config", "/etc/pandora"}

type cliConfig struct {
	Engine     engine.Config    `config:",squash"`
	Log        logConfig        `config:"log"`
	Monitoring monitoringConfig `config:"monitoring"`
}

type logConfig struct {
	Level zapcore.Level `config:"level"`
	File  string        `config:"file"`
}

// TODO(skipor): log sampling with WARN when first message is dropped, and WARN at finish with all
// filtered out entries num. Message is filtered out when zapcore.CoreEnable returns true but
// zapcore.Core.Check return nil.
func newLogger(conf logConfig) *zap.Logger {
	zapConf := zap.NewDevelopmentConfig()
	zapConf.OutputPaths = []string{conf.File}
	zapConf.Level.SetLevel(conf.Level)
	log, err := zapConf.Build(zap.AddCaller())
	if err != nil {
		zap.L().Fatal("Logger build failed", zap.Error(err))
	}
	return log
}

func defaultConfig() *cliConfig {
	return &cliConfig{
		Log: logConfig{
			Level: zap.InfoLevel,
			File:  "stdout",
		},
		Monitoring: monitoringConfig{
			Expvar: &expvarConfig{
				Enabled: false,
				Port:    1234,
			},
			CPUProfile: &cpuprofileConfig{
				Enabled: false,
				File:    "cpuprofile.log",
			},
			MemProfile: &memprofileConfig{
				Enabled: false,
				File:    "memprofile.log",
			},
		},
	}
}

// TODO(skipor): make nice spf13/cobra CLI and integrate it with viper
// TODO(skipor): on special command (help or smth else) print list of available plugins

func Run() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of Pandora: pandora [<config_filename>]\n"+"<config_filename> is './%s.(yaml|json|...)' by default\n", defaultConfigFile)
		flag.PrintDefaults()
	}
	var (
		example bool
		expvar  bool
	)
	flag.BoolVar(&example, "example", false, "print example config to STDOUT and exit")
	flag.BoolVar(&expvar, "expvar", false, "enable expvar service (DEPRECATED, use monitoring config section instead)")
	flag.Parse()

	if expvar {
		fmt.Fprintf(os.Stderr, "-expvar flag is DEPRECATED. Use monitoring config section instead\n")
	}

	if example {
		panic("Not implemented yet")
		// TODO: print example config file content
	}

	conf := readConfig()
	log := newLogger(conf.Log)
	zap.ReplaceGlobals(log)
	zap.RedirectStdLog(log)

	closeMonitoring := startMonitoring(conf.Monitoring)
	defer closeMonitoring()
	m := newEngineMetrics()
	startReport(m)

	pandora := engine.New(log, m, conf.Engine)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errs := make(chan error)
	go runEngine(ctx, pandora, errs)

	sigs := make(chan os.Signal, 2)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	// waiting for signal or error message from engine
	select {
	case sig := <-sigs:
		switch sig {
		case syscall.SIGINT:
			const interruptTimeout = 5 * time.Second
			log.Info("SIGINT received. Trying to stop gracefully.", zap.Duration("timeout", interruptTimeout))
			cancel()
			select {
			case <-time.After(interruptTimeout):
				log.Fatal("Interrupt timeout exceeded")
			case sig := <-sigs:
				log.Fatal("Another signal received. Quiting.", zap.Stringer("signal", sig))
			case err := <-errs:
				log.Fatal("Engine interrupted", zap.Error(err))
			}
		case syscall.SIGTERM:
			log.Fatal("SIGTERM received. Quiting.")
		default:
			log.Fatal("Unexpected signal received. Quiting.", zap.Stringer("signal", sig))
		}
	case err := <-errs:
		switch err {
		case nil:
			log.Info("Pandora engine successfully finished it's work")
		case err:
			const awaitTimeout = 3 * time.Second
			log.Error("Engine run failed. Awaiting started tasks.", zap.Error(err), zap.Duration("timeout", awaitTimeout))
			cancel()
			time.AfterFunc(awaitTimeout, func() {
				log.Fatal("Engine tasks timeout exceeded.")
			})
			pandora.Wait()
			log.Fatal("Engine run failed. Pandora graceful shutdown successfully finished")
		}
	}
	log.Info("Engine run successfully finished")
}

func runEngine(ctx context.Context, engine *engine.Engine, errs chan error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	errs <- engine.Run(ctx)
}

func readConfig() *cliConfig {
	log, err := zap.NewDevelopment(zap.AddCaller())
	if err != nil {
		panic(err)
	}
	log = log.WithOptions(zap.WrapCore(zaputil.NewStackExtractCore))
	zap.ReplaceGlobals(log)
	zap.RedirectStdLog(log)

	v := newViper()

	args := flag.Args()
	if len(args) > 0 {
		switch {
		case len(args) > 1:
			zap.L().Fatal("Too many command line arguments", zap.Strings("args", flag.Args()))
		case args[0] == stdinConfigSelector:
			log.Info("Reading config from standard input")
			useStdinConfig = true
		default:
			v.SetConfigFile(args[0])
		}
	}

	if useStdinConfig {
		v.SetConfigType("yaml")
		configBuffer, err := ioutil.ReadAll(bufio.NewReader(os.Stdin))
		if err != nil {
			log.Fatal("Cannot read from standard input", zap.Error(err))
		}
		err = v.ReadConfig(strings.NewReader(string(configBuffer)))
		if err != nil {
			log.Fatal("Config parsing failed", zap.Error(err))
		}
	} else {
		err = v.ReadInConfig()
		log.Info("Reading config", zap.String("file", v.ConfigFileUsed()))
		if err != nil {
			log.Fatal("Config read failed", zap.Error(err))
		}
	}

	conf := defaultConfig()
	err = config.DecodeAndValidate(v.AllSettings(), conf)
	if err != nil {
		log.Fatal("Config decode failed", zap.Error(err))
	}
	return conf
}

func newViper() *viper.Viper {
	v := viper.New()
	v.SetConfigName(defaultConfigFile)
	for _, dir := range configSearchDirs {
		v.AddConfigPath(dir)
	}
	return v
}

type monitoringConfig struct {
	Expvar     *expvarConfig
	CPUProfile *cpuprofileConfig
	MemProfile *memprofileConfig
}

type expvarConfig struct {
	Enabled bool `config:"enabled"`
	Port    int  `config:"port" validate:"required"`
}

type cpuprofileConfig struct {
	Enabled bool   `config:"enabled"`
	File    string `config:"file"`
}

type memprofileConfig struct {
	Enabled bool   `config:"enabled"`
	File    string `config:"file"`
}

func startMonitoring(conf monitoringConfig) (stop func()) {
	zap.L().Debug("Start monitoring", zap.Reflect("conf", conf))
	if conf.Expvar != nil {
		if conf.Expvar.Enabled {
			go func() {
				err := http.ListenAndServe(":"+strconv.Itoa(conf.Expvar.Port), nil)
				zap.L().Fatal("Monitoring server failed", zap.Error(err))
			}()
		}
	}
	var stops []func()
	if conf.CPUProfile.Enabled {
		f, err := os.Create(conf.CPUProfile.File)
		if err != nil {
			zap.L().Fatal("CPU profile file create fail", zap.Error(err))
		}
		zap.L().Info("Starting CPU profiling")
		pprof.StartCPUProfile(f)
		stops = append(stops, func() {
			pprof.StopCPUProfile()
			f.Close()
		})
	}
	if conf.MemProfile.Enabled {
		f, err := os.Create(conf.MemProfile.File)
		if err != nil {
			zap.L().Fatal("Memory profile file create fail", zap.Error(err))
		}
		stops = append(stops, func() {
			zap.L().Info("Writing memory profile")
			runtime.GC()
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
