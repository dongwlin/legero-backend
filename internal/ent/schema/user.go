package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"github.com/dongwlin/legero-backend/internal/model/types"
)

// User holds the schema definition for the User entity.
type User struct {
	ent.Schema
}

// Fields of the User.
func (User) Fields() []ent.Field {
	return []ent.Field{
		field.String("nickname").
			NotEmpty(),
		field.String("username").
			Unique(),
		field.Enum("role").
			Values(
				types.RoleWaiter.String(),
				types.RoleChef.String(),
				types.RoleManager.String(),
			),
		field.String("password_hash").
			NotEmpty(),
		field.String("phone_number").
			NotEmpty(),
		field.Enum("status").
			Values(
				types.StatusActive.String(),
				types.StatusBlocked.String(),
			).
			Default(types.StatusActive.String()),
		field.Time("blocked_at").
			Optional(),
		field.Time("created_at").
			Default(time.Now),
		field.Time("updated_at").
			Default(time.Now).
			UpdateDefault(time.Now),
	}
}

// Edges of the User.
func (User) Edges() []ent.Edge {
	return []ent.Edge{}
}
