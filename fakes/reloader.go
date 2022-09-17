package fakes

import (
	"sync"

	"github.com/paketo-buildpacks/libreload-packit"
	"github.com/paketo-buildpacks/packit/v2"
)

type Reloader struct {
	ShouldEnableLiveReloadCall struct {
		mutex     sync.Mutex
		CallCount int
		Returns   struct {
			Bool  bool
			Error error
		}
		Stub func() (bool, error)
	}
	TransformReloadableProcessesCall struct {
		mutex     sync.Mutex
		CallCount int
		Receives  struct {
			OriginalProcess packit.Process
			Spec            libreload.ReloadableProcessSpec
		}
		Returns struct {
			NonReloadable packit.Process
			Reloadable    packit.Process
		}
		Stub func(packit.Process, libreload.ReloadableProcessSpec) (packit.Process, packit.Process)
	}
}

func (f *Reloader) ShouldEnableLiveReload() (bool, error) {
	f.ShouldEnableLiveReloadCall.mutex.Lock()
	defer f.ShouldEnableLiveReloadCall.mutex.Unlock()
	f.ShouldEnableLiveReloadCall.CallCount++
	if f.ShouldEnableLiveReloadCall.Stub != nil {
		return f.ShouldEnableLiveReloadCall.Stub()
	}
	return f.ShouldEnableLiveReloadCall.Returns.Bool, f.ShouldEnableLiveReloadCall.Returns.Error
}
func (f *Reloader) TransformReloadableProcesses(param1 packit.Process, param2 libreload.ReloadableProcessSpec) (packit.Process, packit.Process) {
	f.TransformReloadableProcessesCall.mutex.Lock()
	defer f.TransformReloadableProcessesCall.mutex.Unlock()
	f.TransformReloadableProcessesCall.CallCount++
	f.TransformReloadableProcessesCall.Receives.OriginalProcess = param1
	f.TransformReloadableProcessesCall.Receives.Spec = param2
	if f.TransformReloadableProcessesCall.Stub != nil {
		return f.TransformReloadableProcessesCall.Stub(param1, param2)
	}
	return f.TransformReloadableProcessesCall.Returns.NonReloadable, f.TransformReloadableProcessesCall.Returns.Reloadable
}
