package liboc

type DeprecatedNote struct {
	Name              string
	Description       string
	DeprecatedVersion string
	ScheduledVersion  string
	EnvName           string
	MigrationLink     string
}

type DeprecatedManager interface {
	ReportDeprecated(feature DeprecatedNote)
}

type deprecatedManager struct{}

func (m *deprecatedManager) ReportDeprecated(feature DeprecatedNote) {
}