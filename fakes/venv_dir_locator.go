package fakes

import "sync"

type VenvDirLocator struct {
	LocateVenvDirCall struct {
		mutex     sync.Mutex
		CallCount int
		Receives  struct {
			Path string
		}
		Returns struct {
			VenvDir string
			Err     error
		}
		Stub func(string) (string, error)
	}
}

func (f *VenvDirLocator) LocateVenvDir(param1 string) (string, error) {
	f.LocateVenvDirCall.mutex.Lock()
	defer f.LocateVenvDirCall.mutex.Unlock()
	f.LocateVenvDirCall.CallCount++
	f.LocateVenvDirCall.Receives.Path = param1
	if f.LocateVenvDirCall.Stub != nil {
		return f.LocateVenvDirCall.Stub(param1)
	}
	return f.LocateVenvDirCall.Returns.VenvDir, f.LocateVenvDirCall.Returns.Err
}
