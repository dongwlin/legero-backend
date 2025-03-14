package types

type StepStatus string

const (
	StepStatusNone       StepStatus = "none"
	StepStatusUnrequired StepStatus = "unrequired"
	StepStatusNotStarted StepStatus = "not-started"
	StepStatusInProgress StepStatus = "in-progress"
	StepStatusCompleted  StepStatus = "completed"
)

func (s StepStatus) String() string {
	return string(s)
}
