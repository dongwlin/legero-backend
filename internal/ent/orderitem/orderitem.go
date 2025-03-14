// Code generated by ent, DO NOT EDIT.

package orderitem

import (
	"fmt"
	"time"

	"entgo.io/ent/dialect/sql"
)

const (
	// Label holds the string label denoting the orderitem type in the database.
	Label = "order_item"
	// FieldID holds the string denoting the id field in the database.
	FieldID = "id"
	// FieldDailyID holds the string denoting the daily_id field in the database.
	FieldDailyID = "daily_id"
	// FieldIncludeNoodles holds the string denoting the include_noodles field in the database.
	FieldIncludeNoodles = "include_noodles"
	// FieldNoodleType holds the string denoting the noodle_type field in the database.
	FieldNoodleType = "noodle_type"
	// FieldCustomNoodleType holds the string denoting the custom_noodle_type field in the database.
	FieldCustomNoodleType = "custom_noodle_type"
	// FieldNoodleAmount holds the string denoting the noodle_amount field in the database.
	FieldNoodleAmount = "noodle_amount"
	// FieldExtraNoodleBlocks holds the string denoting the extra_noodle_blocks field in the database.
	FieldExtraNoodleBlocks = "extra_noodle_blocks"
	// FieldSize holds the string denoting the size field in the database.
	FieldSize = "size"
	// FieldCustomSizePrice holds the string denoting the custom_size_price field in the database.
	FieldCustomSizePrice = "custom_size_price"
	// FieldMeatAvailable holds the string denoting the meat_available field in the database.
	FieldMeatAvailable = "meat_available"
	// FieldMeatExcluded holds the string denoting the meat_excluded field in the database.
	FieldMeatExcluded = "meat_excluded"
	// FieldGreens holds the string denoting the greens field in the database.
	FieldGreens = "greens"
	// FieldScallion holds the string denoting the scallion field in the database.
	FieldScallion = "scallion"
	// FieldPepper holds the string denoting the pepper field in the database.
	FieldPepper = "pepper"
	// FieldDiningMethod holds the string denoting the dining_method field in the database.
	FieldDiningMethod = "dining_method"
	// FieldPackaging holds the string denoting the packaging field in the database.
	FieldPackaging = "packaging"
	// FieldPackagingMethod holds the string denoting the packaging_method field in the database.
	FieldPackagingMethod = "packaging_method"
	// FieldNote holds the string denoting the note field in the database.
	FieldNote = "note"
	// FieldPrice holds the string denoting the price field in the database.
	FieldPrice = "price"
	// FieldProgressNoodles holds the string denoting the progress_noodles field in the database.
	FieldProgressNoodles = "progress_noodles"
	// FieldProgressMeat holds the string denoting the progress_meat field in the database.
	FieldProgressMeat = "progress_meat"
	// FieldCompletedAt holds the string denoting the completed_at field in the database.
	FieldCompletedAt = "completed_at"
	// FieldCreatedAt holds the string denoting the created_at field in the database.
	FieldCreatedAt = "created_at"
	// Table holds the table name of the orderitem in the database.
	Table = "order_items"
)

// Columns holds all SQL columns for orderitem fields.
var Columns = []string{
	FieldID,
	FieldDailyID,
	FieldIncludeNoodles,
	FieldNoodleType,
	FieldCustomNoodleType,
	FieldNoodleAmount,
	FieldExtraNoodleBlocks,
	FieldSize,
	FieldCustomSizePrice,
	FieldMeatAvailable,
	FieldMeatExcluded,
	FieldGreens,
	FieldScallion,
	FieldPepper,
	FieldDiningMethod,
	FieldPackaging,
	FieldPackagingMethod,
	FieldNote,
	FieldPrice,
	FieldProgressNoodles,
	FieldProgressMeat,
	FieldCompletedAt,
	FieldCreatedAt,
}

// ValidColumn reports if the column name is valid (part of the table columns).
func ValidColumn(column string) bool {
	for i := range Columns {
		if column == Columns[i] {
			return true
		}
	}
	return false
}

var (
	// DefaultIncludeNoodles holds the default value on creation for the "include_noodles" field.
	DefaultIncludeNoodles bool
	// DefaultCustomNoodleType holds the default value on creation for the "custom_noodle_type" field.
	DefaultCustomNoodleType string
	// DefaultExtraNoodleBlocks holds the default value on creation for the "extra_noodle_blocks" field.
	DefaultExtraNoodleBlocks int
	// DefaultCustomSizePrice holds the default value on creation for the "custom_size_price" field.
	DefaultCustomSizePrice float64
	// DefaultMeatAvailable holds the default value on creation for the "meat_available" field.
	DefaultMeatAvailable []string
	// DefaultMeatExcluded holds the default value on creation for the "meat_excluded" field.
	DefaultMeatExcluded []string
	// DefaultNote holds the default value on creation for the "note" field.
	DefaultNote string
	// DefaultPrice holds the default value on creation for the "price" field.
	DefaultPrice float64
	// DefaultCreatedAt holds the default value on creation for the "created_at" field.
	DefaultCreatedAt func() time.Time
)

// NoodleType defines the type for the "noodle_type" enum field.
type NoodleType string

// NoodleType values.
const (
	NoodleTypeNone     NoodleType = "none"
	NoodleTypeFlatRice NoodleType = "flat_rice"
	NoodleTypeThinRice NoodleType = "thin_rice"
	NoodleTypeYi       NoodleType = "yi"
	NoodleTypeCustom   NoodleType = "custom"
)

func (nt NoodleType) String() string {
	return string(nt)
}

// NoodleTypeValidator is a validator for the "noodle_type" field enum values. It is called by the builders before save.
func NoodleTypeValidator(nt NoodleType) error {
	switch nt {
	case NoodleTypeNone, NoodleTypeFlatRice, NoodleTypeThinRice, NoodleTypeYi, NoodleTypeCustom:
		return nil
	default:
		return fmt.Errorf("orderitem: invalid enum value for noodle_type field: %q", nt)
	}
}

// NoodleAmount defines the type for the "noodle_amount" enum field.
type NoodleAmount string

// NoodleAmount values.
const (
	NoodleAmountNone    NoodleAmount = "none"
	NoodleAmountLight   NoodleAmount = "light"
	NoodleAmountRegular NoodleAmount = "regular"
	NoodleAmountHeavy   NoodleAmount = "heavy"
	NoodleAmountExclude NoodleAmount = "exclude"
)

func (na NoodleAmount) String() string {
	return string(na)
}

// NoodleAmountValidator is a validator for the "noodle_amount" field enum values. It is called by the builders before save.
func NoodleAmountValidator(na NoodleAmount) error {
	switch na {
	case NoodleAmountNone, NoodleAmountLight, NoodleAmountRegular, NoodleAmountHeavy, NoodleAmountExclude:
		return nil
	default:
		return fmt.Errorf("orderitem: invalid enum value for noodle_amount field: %q", na)
	}
}

// Size defines the type for the "size" enum field.
type Size string

// Size values.
const (
	SizeNone   Size = "none"
	SizeSmall  Size = "small"
	SizeMiddle Size = "middle"
	SizeLarge  Size = "large"
	SizeCustom Size = "custom"
)

func (s Size) String() string {
	return string(s)
}

// SizeValidator is a validator for the "size" field enum values. It is called by the builders before save.
func SizeValidator(s Size) error {
	switch s {
	case SizeNone, SizeSmall, SizeMiddle, SizeLarge, SizeCustom:
		return nil
	default:
		return fmt.Errorf("orderitem: invalid enum value for size field: %q", s)
	}
}

// Greens defines the type for the "greens" enum field.
type Greens string

// Greens values.
const (
	GreensNone    Greens = "none"
	GreensLight   Greens = "light"
	GreensRegular Greens = "regular"
	GreensHeavy   Greens = "heavy"
	GreensExclude Greens = "exclude"
)

func (gr Greens) String() string {
	return string(gr)
}

// GreensValidator is a validator for the "greens" field enum values. It is called by the builders before save.
func GreensValidator(gr Greens) error {
	switch gr {
	case GreensNone, GreensLight, GreensRegular, GreensHeavy, GreensExclude:
		return nil
	default:
		return fmt.Errorf("orderitem: invalid enum value for greens field: %q", gr)
	}
}

// Scallion defines the type for the "scallion" enum field.
type Scallion string

// Scallion values.
const (
	ScallionNone    Scallion = "none"
	ScallionLight   Scallion = "light"
	ScallionRegular Scallion = "regular"
	ScallionHeavy   Scallion = "heavy"
	ScallionExclude Scallion = "exclude"
)

func (s Scallion) String() string {
	return string(s)
}

// ScallionValidator is a validator for the "scallion" field enum values. It is called by the builders before save.
func ScallionValidator(s Scallion) error {
	switch s {
	case ScallionNone, ScallionLight, ScallionRegular, ScallionHeavy, ScallionExclude:
		return nil
	default:
		return fmt.Errorf("orderitem: invalid enum value for scallion field: %q", s)
	}
}

// Pepper defines the type for the "pepper" enum field.
type Pepper string

// Pepper values.
const (
	PepperNone    Pepper = "none"
	PepperLight   Pepper = "light"
	PepperRegular Pepper = "regular"
	PepperHeavy   Pepper = "heavy"
	PepperExclude Pepper = "exclude"
)

func (pe Pepper) String() string {
	return string(pe)
}

// PepperValidator is a validator for the "pepper" field enum values. It is called by the builders before save.
func PepperValidator(pe Pepper) error {
	switch pe {
	case PepperNone, PepperLight, PepperRegular, PepperHeavy, PepperExclude:
		return nil
	default:
		return fmt.Errorf("orderitem: invalid enum value for pepper field: %q", pe)
	}
}

// DiningMethod defines the type for the "dining_method" enum field.
type DiningMethod string

// DiningMethod values.
const (
	DiningMethodNone    DiningMethod = "none"
	DiningMethodDineIn  DiningMethod = "dine_in"
	DiningMethodTakeOut DiningMethod = "take_out"
)

func (dm DiningMethod) String() string {
	return string(dm)
}

// DiningMethodValidator is a validator for the "dining_method" field enum values. It is called by the builders before save.
func DiningMethodValidator(dm DiningMethod) error {
	switch dm {
	case DiningMethodNone, DiningMethodDineIn, DiningMethodTakeOut:
		return nil
	default:
		return fmt.Errorf("orderitem: invalid enum value for dining_method field: %q", dm)
	}
}

// Packaging defines the type for the "packaging" enum field.
type Packaging string

// Packaging values.
const (
	PackagingNone        Packaging = "none"
	PackagingPlasticBox  Packaging = "plastic_box"
	PackagingPlasticBag  Packaging = "plastic_bag"
	PackagingCustomerOwn Packaging = "customer_own"
)

func (pa Packaging) String() string {
	return string(pa)
}

// PackagingValidator is a validator for the "packaging" field enum values. It is called by the builders before save.
func PackagingValidator(pa Packaging) error {
	switch pa {
	case PackagingNone, PackagingPlasticBox, PackagingPlasticBag, PackagingCustomerOwn:
		return nil
	default:
		return fmt.Errorf("orderitem: invalid enum value for packaging field: %q", pa)
	}
}

// PackagingMethod defines the type for the "packaging_method" enum field.
type PackagingMethod string

// PackagingMethod values.
const (
	PackagingMethodNone          PackagingMethod = "none"
	PackagingMethodCombined      PackagingMethod = "combined"
	PackagingMethodNoodleSoupSep PackagingMethod = "noodle_soup_sep"
)

func (pm PackagingMethod) String() string {
	return string(pm)
}

// PackagingMethodValidator is a validator for the "packaging_method" field enum values. It is called by the builders before save.
func PackagingMethodValidator(pm PackagingMethod) error {
	switch pm {
	case PackagingMethodNone, PackagingMethodCombined, PackagingMethodNoodleSoupSep:
		return nil
	default:
		return fmt.Errorf("orderitem: invalid enum value for packaging_method field: %q", pm)
	}
}

// ProgressNoodles defines the type for the "progress_noodles" enum field.
type ProgressNoodles string

// ProgressNoodles values.
const (
	ProgressNoodlesNone       ProgressNoodles = "none"
	ProgressNoodlesUnrequired ProgressNoodles = "unrequired"
	ProgressNoodlesNotStarted ProgressNoodles = "not-started"
	ProgressNoodlesInProgress ProgressNoodles = "in-progress"
	ProgressNoodlesCompleted  ProgressNoodles = "completed"
)

func (pn ProgressNoodles) String() string {
	return string(pn)
}

// ProgressNoodlesValidator is a validator for the "progress_noodles" field enum values. It is called by the builders before save.
func ProgressNoodlesValidator(pn ProgressNoodles) error {
	switch pn {
	case ProgressNoodlesNone, ProgressNoodlesUnrequired, ProgressNoodlesNotStarted, ProgressNoodlesInProgress, ProgressNoodlesCompleted:
		return nil
	default:
		return fmt.Errorf("orderitem: invalid enum value for progress_noodles field: %q", pn)
	}
}

// ProgressMeat defines the type for the "progress_meat" enum field.
type ProgressMeat string

// ProgressMeat values.
const (
	ProgressMeatNone       ProgressMeat = "none"
	ProgressMeatUnrequired ProgressMeat = "unrequired"
	ProgressMeatNotStarted ProgressMeat = "not-started"
	ProgressMeatInProgress ProgressMeat = "in-progress"
	ProgressMeatCompleted  ProgressMeat = "completed"
)

func (pm ProgressMeat) String() string {
	return string(pm)
}

// ProgressMeatValidator is a validator for the "progress_meat" field enum values. It is called by the builders before save.
func ProgressMeatValidator(pm ProgressMeat) error {
	switch pm {
	case ProgressMeatNone, ProgressMeatUnrequired, ProgressMeatNotStarted, ProgressMeatInProgress, ProgressMeatCompleted:
		return nil
	default:
		return fmt.Errorf("orderitem: invalid enum value for progress_meat field: %q", pm)
	}
}

// OrderOption defines the ordering options for the OrderItem queries.
type OrderOption func(*sql.Selector)

// ByID orders the results by the id field.
func ByID(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldID, opts...).ToFunc()
}

// ByDailyID orders the results by the daily_id field.
func ByDailyID(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldDailyID, opts...).ToFunc()
}

// ByIncludeNoodles orders the results by the include_noodles field.
func ByIncludeNoodles(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldIncludeNoodles, opts...).ToFunc()
}

// ByNoodleType orders the results by the noodle_type field.
func ByNoodleType(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldNoodleType, opts...).ToFunc()
}

// ByCustomNoodleType orders the results by the custom_noodle_type field.
func ByCustomNoodleType(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldCustomNoodleType, opts...).ToFunc()
}

// ByNoodleAmount orders the results by the noodle_amount field.
func ByNoodleAmount(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldNoodleAmount, opts...).ToFunc()
}

// ByExtraNoodleBlocks orders the results by the extra_noodle_blocks field.
func ByExtraNoodleBlocks(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldExtraNoodleBlocks, opts...).ToFunc()
}

// BySize orders the results by the size field.
func BySize(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldSize, opts...).ToFunc()
}

// ByCustomSizePrice orders the results by the custom_size_price field.
func ByCustomSizePrice(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldCustomSizePrice, opts...).ToFunc()
}

// ByGreens orders the results by the greens field.
func ByGreens(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldGreens, opts...).ToFunc()
}

// ByScallion orders the results by the scallion field.
func ByScallion(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldScallion, opts...).ToFunc()
}

// ByPepper orders the results by the pepper field.
func ByPepper(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldPepper, opts...).ToFunc()
}

// ByDiningMethod orders the results by the dining_method field.
func ByDiningMethod(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldDiningMethod, opts...).ToFunc()
}

// ByPackaging orders the results by the packaging field.
func ByPackaging(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldPackaging, opts...).ToFunc()
}

// ByPackagingMethod orders the results by the packaging_method field.
func ByPackagingMethod(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldPackagingMethod, opts...).ToFunc()
}

// ByNote orders the results by the note field.
func ByNote(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldNote, opts...).ToFunc()
}

// ByPrice orders the results by the price field.
func ByPrice(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldPrice, opts...).ToFunc()
}

// ByProgressNoodles orders the results by the progress_noodles field.
func ByProgressNoodles(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldProgressNoodles, opts...).ToFunc()
}

// ByProgressMeat orders the results by the progress_meat field.
func ByProgressMeat(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldProgressMeat, opts...).ToFunc()
}

// ByCompletedAt orders the results by the completed_at field.
func ByCompletedAt(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldCompletedAt, opts...).ToFunc()
}

// ByCreatedAt orders the results by the created_at field.
func ByCreatedAt(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldCreatedAt, opts...).ToFunc()
}
