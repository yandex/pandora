package cli

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"runtime/pprof"
	"time"

	"github.com/yandex/pandora/config"
	"github.com/yandex/pandora/engine"
	"github.com/yandex/pandora/utils"
)

const Version = "0.1.2"

func Run() {
	fmt.Printf("Pandora v%s\n", Version)
	flag.Usage = func() {
		fmt.Fprint(os.Stderr, "Usage of Pandora: pandora [<config_filename>]\n"+
			"<config_filename> is './load.json' by default\n")
		flag.PrintDefaults()
	}
	example := flag.Bool("example", false, "print example config to STDOUT and exit")
	cpuprofile := flag.String("cpuprofile", "", "write cpu profile to file")
	memprofile := flag.String("memprofile", "", "write memory profile to this file")
	expvarHttp := flag.Bool("expvar", false, "start HTTP server with monitoring variables")
	flag.Parse()

	if *example {
		fmt.Println(exampleConfig)
		return
	}

	if *expvarHttp {
		go http.ListenAndServe(":1234", nil)
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
		defer func() {
			pprof.StopCPUProfile()
			f.Close()
		}()
	}
	if *memprofile != "" {
		defer func() {
			f, err := os.Create(*memprofile)
			if err != nil {
				log.Fatal(err)
			}
			pprof.WriteHeapProfile(f)
			f.Close()
		}()
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
