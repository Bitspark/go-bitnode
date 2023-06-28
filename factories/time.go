package factories

import (
	"fmt"
	"github.com/Bitspark/go-bitnode/bitnode"
	"time"
)

// Time factory.

type TimeFactory struct {
}

var _ bitnode.Factory = &TimeFactory{}

func NewTimeFactory() *TimeFactory {
	return &TimeFactory{}
}

func (f *TimeFactory) Name() string {
	return "time"
}

func (f *TimeFactory) Implementation(impl bitnode.Implementation) (bitnode.Implementation, error) {
	if impl == nil {
		return &TimeImpl{}, nil
	}
	nImpl, ok := impl.(*TimeImpl)
	if !ok {
		return nil, fmt.Errorf("not a time implementation")
	}
	return nImpl, nil
}

// Time implementation.

type TimeImpl struct {
	System string `json:"system" yaml:"system"`
	creds  bitnode.Credentials
}

var _ bitnode.Implementation = &TimeImpl{}

func (m *TimeImpl) Implement(node *bitnode.NativeNode, sys bitnode.System) error {
	sys.AddExtension("time", &timeImpl{m: m})

	if m.System == "Clock" {
		// Hubs

		getTimestamp := sys.GetHub("getTimestamp")
		getTimestamp.Handle(bitnode.NewNativeFunction(func(creds bitnode.Credentials, vals ...bitnode.HubItem) ([]bitnode.HubItem, error) {
			ts := float64(time.Now().UnixMicro()) / float64(time.Second/time.Microsecond)
			return []bitnode.HubItem{ts}, nil
		}))

		// Status and message

		sys.LogInfo("Clock running")
		sys.SetStatus(bitnode.SystemStatusRunning)

		return nil
	}
	if m.System == "Trigger" {
		sys.AddCallback(bitnode.LifecycleCreate, bitnode.NewNativeEvent(func(vals ...bitnode.HubItem) error {
			if len(vals) == 0 {
				return fmt.Errorf("require interval")
			}

			tick := sys.GetHub("tick")
			interval := vals[0].(float64)
			start := time.Now()
			ticks := int64(0)
			go func() {
				for {
					timePassed := float64(time.Now().Sub(start).Milliseconds()) / float64(time.Second/time.Millisecond)
					tick.Emit("", map[string]any{"ticks": ticks, "elapsed": timePassed})
					time.Sleep(time.Duration(interval * float64(time.Second)))
					ticks++
				}
			}()

			return nil
		}))

		// Status and message

		sys.LogInfo("Trigger running")
		sys.SetStatus(bitnode.SystemStatusRunning)

		return nil
	}
	return nil
}

func (m *TimeImpl) Extend(node *bitnode.NativeNode, ext bitnode.Implementation) (bitnode.Implementation, error) {
	panic("implement me")
}

func (m *TimeImpl) ToInterface() (any, error) {
	return nil, nil
}

func (m *TimeImpl) FromInterface(i any) error {
	ti := i.(map[string]any)
	m.System = ti["system"].(string)
	return nil
}

func (m *TimeImpl) Validate() error {
	panic("implement me")
}

type timeImpl struct {
	m *TimeImpl
}

var _ bitnode.SystemExtension = &timeImpl{}

func (h *timeImpl) Implementation() bitnode.Implementation {
	return h.m
}
