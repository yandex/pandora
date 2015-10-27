package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
)

func PandoraRunConfig(gc *GlobalConfig) {
	pools := make([]*UserPool, 0, len(gc.Pools))
	for _, upc := range gc.Pools {
		up, err := NewUserPoolFromConfig(&upc)
		if err != nil {
			log.Printf("Could not create user pool: %s", err)
			continue
		}
		pools = append(pools, up)
	}
	for _, up := range pools {
		up.Start()
	}
	for _, up := range pools {
		<-up.done
	}

	log.Println("Done")
}

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
	c, err := NewConfigFromJson(jsonDoc)
	if err != nil {
		log.Printf("Could not unmarshal config from json: %s", err)
		return
	}
	PandoraRunConfig(c)
}

// func (ps *PhoutSample) String() string {
// 	return fmt.Sprintf(
// 		"%.3f\t%s\t%d\t"+
// 			"%d\t%d\t"+
// 			"%d\t%d\t"+
// 			"%d\t"+
// 			"%d\t%d\t"+
// 			"%d\t%d",
// 		ps.ts, ps.tag, ps.rt,
// 		ps.connect, ps.send,
// 		ps.latency, ps.receive,
// 		ps.interval_event,
// 		ps.egress, ps.igress,
// 		ps.netCode, ps.protoCode,
// 	)
// }
