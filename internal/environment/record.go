package environment

type Record struct {
	ProfileSHA256       string `json:"profile_sha256,omitempty"`
	PreparedEnvironment string `json:"prepared_environment_id,omitempty"`
	Runtime             string `json:"runtime"`
	WorkspaceTarget     string `json:"workspace_target,omitempty"`
	UID                 int    `json:"uid,omitempty"`
	GID                 int    `json:"gid,omitempty"`
	SSH                 string `json:"ssh,omitempty"`
	TmpSize             string `json:"tmp_size,omitempty"`
	TmpExecutable       bool   `json:"tmp_executable,omitempty"`
}

func RecordFor(doc Document, manifest Manifest) Record {
	p := doc.Profile
	return Record{
		ProfileSHA256: doc.SHA256, PreparedEnvironment: manifest.PreparedEnvironmentID,
		Runtime: p.Runtime.Driver, WorkspaceTarget: p.Runtime.WorkspaceTarget,
		UID: p.RootFS.User.UID, GID: p.RootFS.User.GID,
		SSH: p.Runtime.Credentials.SSH, TmpSize: p.Runtime.Tmp.Size,
		TmpExecutable: p.Runtime.Tmp.Executable,
	}
}

func CurrentManifest(store, name string) (Manifest, error) {
	return currentManifest(store, name)
}
