package types

type Status string

const (
	StatusActive  Status = "active"
	StatusBlocked Status = "blocked"
)

func (s Status) String() string {
	return string(s)
}
