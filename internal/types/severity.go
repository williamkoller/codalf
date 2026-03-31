package types

type Severity string

const (
	SeverityCritical Severity = "critical"
	SeverityWarning  Severity = "warning"
	SeverityInfo     Severity = "info"
)

func (s Severity) IsValid() bool {
	switch s {
	case SeverityCritical, SeverityWarning, SeverityInfo:
		return true
	}
	return false
}
