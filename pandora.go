package main

import (
	"log"
)

type PhoutSample struct {
	ts             float64
	tag            string
	rt             int
	connect        int
	send           int
	latency        int
	receive        int
	interval_event int
	egress         int
	igress         int
	netCode        int
	protoCode      int
}

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
	log.Println("Done")
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
