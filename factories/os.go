package factories

import (
	"fmt"
	"github.com/Bitspark/go-bitnode/bitnode"
	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/host"
	"github.com/shirou/gopsutil/mem"
	"time"
)

// OS factory.

type OSFactory struct {
}

var _ bitnode.Factory = &OSFactory{}

func NewOSFactory() *OSFactory {
	return &OSFactory{}
}

func (f *OSFactory) Name() string {
	return "os"
}

func (f *OSFactory) Implementation(impl bitnode.Implementation) (bitnode.Implementation, error) {
	if impl == nil {
		return &OSImpl{}, nil
	}
	nImpl, ok := impl.(*OSImpl)
	if !ok {
		return nil, fmt.Errorf("not an OS implementation")
	}
	return nImpl, nil
}

// OS implementation.

type OSImpl struct {
}

var _ bitnode.Implementation = &OSImpl{}

func (m *OSImpl) Implement(node *bitnode.NativeNode, sys bitnode.System) error {
	sys.AddExtension("os", &osExtension{m: m})

	sys.LogInfo("Implementing OperatingSystem")

	sys.AddCallback(bitnode.LifecycleLoad, bitnode.NewNativeEvent(func(vals ...bitnode.HubItem) error {
		hostnameHub := sys.GetHub("hostname")
		memoryHub := sys.GetHub("memory")
		cpuHub := sys.GetHub("cpus")

		// HOST

		hostStat, _ := host.Info()

		if err := hostnameHub.Set("", hostStat.Hostname); err != nil {
			sys.LogError(err)
		}

		// CPU

		cpuStats, _ := cpu.Info()
		cpus := []bitnode.HubItem{}

		for _, cpuStat := range cpuStats {
			cores := int(cpuStat.Cores)

			cp := map[string]bitnode.HubItem{
				"cpu":   cpuStat.CPU,
				"cores": cores,
				"model": cpuStat.ModelName,
			}

			cpus = append(cpus, cp)
		}

		if err := cpuHub.Set("", cpus); err != nil {
			sys.LogError(err)
		}

		// MEMORY

		go func() {
			for {
				vmStat, _ := mem.VirtualMemory()

				memory := map[string]bitnode.HubItem{
					"total": vmStat.Total,
					"free":  vmStat.Free,
				}
				if err := memoryHub.Set("", memory); err != nil {
					sys.LogError(err)
				}

				time.Sleep(1 * time.Second)
			}
		}()

		sys.LogInfo("OperatingSystem running")
		sys.SetStatus(bitnode.SystemStatusRunning)

		return nil
	}))

	return nil
}

func (m *OSImpl) Extend(node *bitnode.NativeNode, ext bitnode.Implementation) (bitnode.Implementation, error) {
	panic("implement me")
}

func (m *OSImpl) ToInterface() (any, error) {
	return nil, nil
}

func (m *OSImpl) FromInterface(i any) error {
	return nil
}

func (m *OSImpl) Validate() error {
	panic("implement me")
}

type osExtension struct {
	m *OSImpl
}

var _ bitnode.SystemExtension = &osExtension{}

func (h *osExtension) Implementation() bitnode.Implementation {
	return h.m
}
