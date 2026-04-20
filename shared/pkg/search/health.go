package search

type HealthStatus struct {
	Status  string
	Version VersionInfo
}

type VersionInfo struct {
	CommitSHA  string
	CommitDate string
	PkgVersion string
}
