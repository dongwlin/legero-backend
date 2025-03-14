// Code generated by ent, DO NOT EDIT.

package ent

import (
	"context"

	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/dialect/sql/sqlgraph"
	"entgo.io/ent/schema/field"
	"github.com/dongwlin/legero-backend/internal/ent/orderitem"
	"github.com/dongwlin/legero-backend/internal/ent/predicate"
)

// OrderItemDelete is the builder for deleting a OrderItem entity.
type OrderItemDelete struct {
	config
	hooks    []Hook
	mutation *OrderItemMutation
}

// Where appends a list predicates to the OrderItemDelete builder.
func (oid *OrderItemDelete) Where(ps ...predicate.OrderItem) *OrderItemDelete {
	oid.mutation.Where(ps...)
	return oid
}

// Exec executes the deletion query and returns how many vertices were deleted.
func (oid *OrderItemDelete) Exec(ctx context.Context) (int, error) {
	return withHooks(ctx, oid.sqlExec, oid.mutation, oid.hooks)
}

// ExecX is like Exec, but panics if an error occurs.
func (oid *OrderItemDelete) ExecX(ctx context.Context) int {
	n, err := oid.Exec(ctx)
	if err != nil {
		panic(err)
	}
	return n
}

func (oid *OrderItemDelete) sqlExec(ctx context.Context) (int, error) {
	_spec := sqlgraph.NewDeleteSpec(orderitem.Table, sqlgraph.NewFieldSpec(orderitem.FieldID, field.TypeUint64))
	if ps := oid.mutation.predicates; len(ps) > 0 {
		_spec.Predicate = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	affected, err := sqlgraph.DeleteNodes(ctx, oid.driver, _spec)
	if err != nil && sqlgraph.IsConstraintError(err) {
		err = &ConstraintError{msg: err.Error(), wrap: err}
	}
	oid.mutation.done = true
	return affected, err
}

// OrderItemDeleteOne is the builder for deleting a single OrderItem entity.
type OrderItemDeleteOne struct {
	oid *OrderItemDelete
}

// Where appends a list predicates to the OrderItemDelete builder.
func (oido *OrderItemDeleteOne) Where(ps ...predicate.OrderItem) *OrderItemDeleteOne {
	oido.oid.mutation.Where(ps...)
	return oido
}

// Exec executes the deletion query.
func (oido *OrderItemDeleteOne) Exec(ctx context.Context) error {
	n, err := oido.oid.Exec(ctx)
	switch {
	case err != nil:
		return err
	case n == 0:
		return &NotFoundError{orderitem.Label}
	default:
		return nil
	}
}

// ExecX is like Exec, but panics if an error occurs.
func (oido *OrderItemDeleteOne) ExecX(ctx context.Context) {
	if err := oido.Exec(ctx); err != nil {
		panic(err)
	}
}
