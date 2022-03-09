package fakes

import (
	"sync"

	"github.com/paketo-buildpacks/packit"
)

type InstallProcess struct {
	ExecuteCall struct {
		mutex     sync.Mutex
		CallCount int
		Receives  struct {
			WorkingDir  string
			TargetLayer packit.Layer
			CacheLayer  packit.Layer
		}
		Returns struct {
			Error error
		}
		Stub func(string, packit.Layer, packit.Layer) error
	}
}

func (f *InstallProcess) Execute(param1 string, param2 packit.Layer, param3 packit.Layer) error {
	f.ExecuteCall.mutex.Lock()
	defer f.ExecuteCall.mutex.Unlock()
	f.ExecuteCall.CallCount++
	f.ExecuteCall.Receives.WorkingDir = param1
	f.ExecuteCall.Receives.TargetLayer = param2
	f.ExecuteCall.Receives.CacheLayer = param3
	if f.ExecuteCall.Stub != nil {
		return f.ExecuteCall.Stub(param1, param2, param3)
	}
	return f.ExecuteCall.Returns.Error
}
