package registry

type ImageConfig struct {
	RepoName string
	RepoTag  string
	SavePath string
}

type Manifest struct {
	Config        ManifestConfig  `json:"config,omitempty"`
	Layers        []ManifestLayer `json:"layers,omitempty"`
	MediaType     string          `json:"mediaType,omitempty"`
	SchemaVersion int             `json:"schemaVersion"`
}

type ManifestConfig struct {
	Digest    string `json:"digest"`
	MediaType string `json:"mediaType"`
	Size      int    `json:"size"`
}

type ManifestLayer struct {
	Digest    string `json:"digest"`
	MediaType string `json:"mediaType"`
	Size      int    `json:"size"`
}

type ManifestList struct {
	Manifests     []ManifestInfo `json:"manifests,omitempty"`
	MediaType     string         `json:"mediaType,omitempty"`
	SchemaVersion int            `json:"schemaVersion"`
}

type ManifestInfo struct {
	Digest    string       `json:"digest"`
	MediaType string       `json:"mediaType"`
	Size      int          `json:"size"`
	Platform  PlatformInfo `json:"platform"`
}

type PlatformInfo struct {
	Architecture string `json:"architecture"`
	OS           string `json:"os"`
}
