package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"time"

	"golang.org/x/net/context"

	"github.com/yandex/pandora/engine"
	"github.com/yandex/pandora/utils"
)

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of Pandora: pandora [<config_filename>]\n"+
			"<config_filename> is './load.json' by default\n")
		flag.PrintDefaults()
	}
	example := flag.Bool("example", false, "print example config to STDOUT and exit")
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
	cfg, err := engine.NewConfigFromJson(jsonDoc)
	if err != nil {
		log.Printf("Could not unmarshal config from json: %s", err)
		return
	}
//	PandoraRunConfig(cfg)

	pandora := engine.New(cfg)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	promise := utils.Promise(func() error { return pandora.Serve(ctx) })

	select {
	case <-utils.NotifyInterrupt():
		log.Print("Interrupting by signal, trying to stop")
		cancel()
		select {
		case err = <- promise:
		case <-time.After(time.Second * 20):
			err = fmt.Errorf("Timeout exceeded")
		}
	case err = <-promise:
	}
	if err != nil {
		log.Fatal(err)
	}
}
