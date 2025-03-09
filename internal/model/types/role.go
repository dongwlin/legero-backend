package types

type Role string

const (
	RoleWaiter  Role = "waiter"
	RoleChef    Role = "chef"
	RoleManager Role = "manager"
)

func (r Role) String() string {
	return string(r)
}
