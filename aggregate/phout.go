package aggregate

import (
	"fmt"
	"os"

	"github.com/yandex/pandora/config"
	"golang.org/x/net/context"
)

type PhoutSample struct {
	TS            float64
	Tag           string
	RT            int
	Connect       int
	Send          int
	Latency       int
	Receive       int
	IntervalEvent int
	Egress        int
	Igress        int
	NetCode       int
	ProtoCode     int
}

func (ps *PhoutSample) String() string {
	return fmt.Sprintf(
		"%.3f\t%s\t%d\t"+
			"%d\t%d\t"+
			"%d\t%d\t"+
			"%d\t"+
			"%d\t%d\t"+
			"%d\t%d",
		ps.TS, ps.Tag, ps.RT,
		ps.Connect, ps.Send,
		ps.Latency, ps.Receive,
		ps.IntervalEvent,
		ps.Egress, ps.Igress,
		ps.NetCode, ps.ProtoCode,
	)
}

type PhantomCompatible interface {
	Sample
	PhoutSample() *PhoutSample
}

type PhoutResultListener struct {
	resultListener

	source <-chan Sample
	phout  *os.File
}

func (rl *PhoutResultListener) handle(r Sample) error {
	pc, ok := r.(PhantomCompatible)
	if ok {
		_, err := rl.phout.WriteString(fmt.Sprintf("%s\n", pc.PhoutSample()))
		if err != nil {
			return err
		}
	} else {
		return fmt.Errorf("Not phantom compatible sample")
	}
	return nil
}

func (rl *PhoutResultListener) Start(ctx context.Context) error {
loop:
	for {
		select {
		case r := <-rl.source:
			rl.handle(r)
		case <-ctx.Done():
			// Context is done, but we should read all data from source
			for {
				select {
				case r := <-rl.source:
					rl.handle(r)
				default:
					break loop
				}
			}
		}
	}
	return nil
}

func NewPhoutResultListener(filename string) (rl ResultListener, err error) {
	var phoutFile *os.File
	if filename == "" {
		phoutFile = os.Stdout
	} else {
		phoutFile, err = os.OpenFile(filename, os.O_APPEND|os.O_WRONLY|os.O_CREATE|os.O_SYNC, 0666)
	}
	ch := make(chan Sample, 32)
	return &PhoutResultListener{
		source: ch,
		resultListener: resultListener{
			sink: ch,
		},
		phout: phoutFile,
	}, nil
}

type phoutResultListeners map[string]ResultListener

func (prls phoutResultListeners) get(c *config.ResultListener) (ResultListener, error) {
	rl, ok := prls[c.Destination]
	if !ok {
		rl, err := NewPhoutResultListener(c.Destination)
		if err != nil {
			return nil, err
		}
		prls[c.Destination] = rl
		return rl, nil
	}
	return rl, nil
}

var defaultPhoutResultListeners = phoutResultListeners{}

// GetPhoutResultListener is not thread safe.
func GetPhoutResultListener(c *config.ResultListener) (ResultListener, error) {
	return defaultPhoutResultListeners.get(c)
}
