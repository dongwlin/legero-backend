package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"github.com/dongwlin/legero-backend/internal/model/types"
)

// OrderItem holds the schema definition for the OrderItem entity.
type OrderItem struct {
	ent.Schema
}

// Fields of the OrderItem.
func (OrderItem) Fields() []ent.Field {
	return []ent.Field{
		field.Uint64("daily_id"),
		field.Bool("include_noodles").
			Default(true),
		field.Enum("noodle_type").
			Values(
				types.NoodleNone.String(),
				types.NoodleFlatRice.String(),
				types.NoodleThinRice.String(),
				types.NoodleYi.String(),
				types.NoodleCustom.String(),
			),
		field.String("custom_noodle_type").
			Default(""),
		field.Enum("noodle_amount").
			Values(
				types.AdjustmentNone.String(),
				types.AdjustmentLight.String(),
				types.AdjustmentRegular.String(),
				types.AdjustmentHeavy.String(),
				types.AdjustmentExclude.String(),
			),
		field.Int("extra_noodle_blocks").
			Default(0),
		field.Enum("size").
			Values(
				types.SizeNone.String(),
				types.SizeSmall.String(),
				types.SizeMiddle.String(),
				types.SizeLarge.String(),
				types.SizeCustom.String(),
			),
		field.Float("custom_size_price").
			Default(0),
		field.Strings("meat_available").
			Default([]string{}),
		field.Strings("meat_excluded").
			Default([]string{}),
		field.Enum("greens").
			Values(
				types.AdjustmentNone.String(),
				types.AdjustmentLight.String(),
				types.AdjustmentRegular.String(),
				types.AdjustmentHeavy.String(),
				types.AdjustmentExclude.String(),
			),
		field.Enum("scallion").
			Values(
				types.AdjustmentNone.String(),
				types.AdjustmentLight.String(),
				types.AdjustmentRegular.String(),
				types.AdjustmentHeavy.String(),
				types.AdjustmentExclude.String(),
			),
		field.Enum("pepper").
			Values(
				types.AdjustmentNone.String(),
				types.AdjustmentLight.String(),
				types.AdjustmentRegular.String(),
				types.AdjustmentHeavy.String(),
				types.AdjustmentExclude.String(),
			),
		field.Enum("dining_method").
			Values(
				types.DiningMethodNone.String(),
				types.DiningMethodDineIn.String(),
				types.DiningMethodTakeOut.String(),
			),
		field.Enum("packaging").
			Values(
				types.PackagingNone.String(),
				types.PackagingPlasticBox.String(),
				types.PackagingPlasticBag.String(),
				types.PackagingCustomerOwn.String(),
			),
		field.Enum("packaging_method").
			Values(
				types.PackagingMethodNone.String(),
				types.PackagingMethodCombined.String(),
				types.PackagingMethodNoodleSoupSep.String(),
			),
		field.String("note").
			Default(""),
		field.Float("price").
			Default(0),
		field.Enum("progress_noodles").
			Values(
				types.StepStatusNone.String(),
				types.StepStatusUnrequired.String(),
				types.StepStatusNotStarted.String(),
				types.StepStatusInProgress.String(),
				types.StepStatusCompleted.String(),
			),
		field.Enum("progress_meat").
			Values(
				types.StepStatusNone.String(),
				types.StepStatusUnrequired.String(),
				types.StepStatusNotStarted.String(),
				types.StepStatusInProgress.String(),
				types.StepStatusCompleted.String(),
			),
		field.Time("completed_at").
			Optional(),
		field.Time("created_at").
			Default(time.Now),
	}
}

// Edges of the OrderItem.
func (OrderItem) Edges() []ent.Edge {
	return nil
}
