package enode

import (
	"errors"
	"path/filepath"

	execenv "github.com/taeels/enode/internal/environment"
)

// LoadExecutionEnvironment는 node-local binding과 공유 profile을 결합한다.
// node config는 자리만 정하고 준비 동작은 profile의 typed field가 정한다.
func LoadExecutionEnvironment(configPath string, local Local) (execenv.Document, execenv.Binding, error) {
	if local.Environment == nil {
		return execenv.Document{}, execenv.Binding{}, errors.New("node config has no environment binding")
	}
	profile := local.Environment.Profile
	if !filepath.IsAbs(profile) {
		profile = filepath.Join(filepath.Dir(configPath), profile)
	}
	doc, err := execenv.Load(profile)
	if err != nil {
		return execenv.Document{}, execenv.Binding{}, err
	}
	return doc, execenv.Binding{
		Store: local.Environment.Store, Scratch: local.Environment.Scratch,
		Workspace: local.Workspace, SSHDir: local.Credentials.SSHDir,
	}, nil
}
