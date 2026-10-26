package model

import (
	"context"
	"errors"
	"fmt"
	"math/big"

	"github.com/uptrace/bun"
	"github.com/veraison/corim-store/pkg/util"
	"github.com/veraison/corim/comid"
)

type IntMatcher struct {
	bun.BaseModel `bun:"table:int_matchers,alias:im"`

	ID     int64 `bun:",pk,autoincrement"`
	Type   string
	Values []*IntMatcherElement `bun:"rel:has-many,join:id=matcher_id"`
}

func NewIntMatcherFromCoRIM(origin *comid.NumberMatcher) (*IntMatcher, error) {
	var matcher IntMatcher
	if err := matcher.FromCoRIM(origin); err != nil {
		return nil, err
	}

	return &matcher, nil
}

func SelectIntMatcher(ctx context.Context, db bun.IDB, id int64) (*IntMatcher, error) {
	ret := IntMatcher{ID: id}

	if err := ret.Select(ctx, db); err != nil {
		return nil, err
	}

	return &ret, nil
}

func (o *IntMatcher) DbID() int64 {
	return o.ID
}

func (o *IntMatcher) TableName() string {
	return "int_matchers"
}

func (o *IntMatcher) IsTable() bool {
	return true
}

func (o *IntMatcher) Insert(ctx context.Context, db bun.IDB) error {
	if _, err := db.NewInsert().Model(o).Exec(ctx); err != nil {
		return err
	}

	for i, value := range o.Values {
		value.MatcherID = o.ID

		if err := value.Insert(ctx, db); err != nil {
			return fmt.Errorf("int matcher value at index %d: %w", i, err)
		}
	}

	return nil
}

func (o *IntMatcher) Select(ctx context.Context, db bun.IDB) error {
	if o.ID == 0 {
		return errors.New("ID not set")
	}

	err := db.NewSelect().
		Model(o).
		Relation("Values").
		Where("im.id = ?", o.ID).
		Scan(ctx)

	if err != nil {
		return err
	}

	return nil
}

func (o *IntMatcher) Delete(ctx context.Context, db bun.IDB) error { // nolint:dupl
	if o.ID == 0 {
		return errors.New("ID not set")
	}

	for i, value := range o.Values {
		if err := value.Delete(ctx, db); err != nil {
			return fmt.Errorf("int matcher value at index %d: %w", i, err)
		}
	}

	_, err := db.NewDelete().Model(o).WherePK().Exec(ctx)
	return err
}

func (o *IntMatcher) ToCoRIM() (*comid.NumberMatcher, error) {
	numbers := make([]comid.Number, len(o.Values))
	for i, element := range o.Values {
		ret := new(big.Int).SetUint64(element.Value)
		if element.Sign < 0 {
			ret.Neg(ret)
		}

		numbers[i] = comid.MustNumberFromBig(ret)
	}

	return matcherFromNumbers(o.Type, numbers)
}

func (o *IntMatcher) FromCoRIM(origin *comid.NumberMatcher) error {
	var values []*IntMatcherElement // nolint:prealloc
	for i, originValue := range util.Enumerate(origin.IterValues()) {
		if !originValue.IsInt() {
			return fmt.Errorf("float value at index %d", i)
		}

		bigInt := originValue.BigInt()
		values = append(values, &IntMatcherElement{
			Sign:  bigInt.Sign(),
			Value: bigInt.Abs(bigInt).Uint64(),
		})
	}

	typ, err := identifyNumberMatcherType(origin)
	if err != nil {
		return err
	}

	o.Type = typ
	o.Values = values

	return nil
}

type IntMatcherElement struct {
	bun.BaseModel `bun:"table:int_matcher_elements,alias:ime"`

	ID        int64 `bun:",pk,autoincrement"`
	Sign      int
	Value     uint64
	MatcherID int64
}

func (o *IntMatcherElement) DbID() int64 {
	return o.ID
}

func (o *IntMatcherElement) TableName() string {
	return "int_matcher_elements"
}

func (o *IntMatcherElement) IsTable() bool {
	return true
}

func (o *IntMatcherElement) Insert(ctx context.Context, db bun.IDB) error {
	_, err := db.NewInsert().Model(o).Exec(ctx)
	return err
}

func (o *IntMatcherElement) Select(ctx context.Context, db bun.IDB) error {
	if o.ID == 0 {
		return errors.New("ID not set")
	}

	return db.NewSelect().Model(o).Where("ime.id = ?", o.ID).Scan(ctx)
}

func (o *IntMatcherElement) Delete(ctx context.Context, db bun.IDB) error { // nolint:dupl
	if o.ID == 0 {
		return errors.New("ID not set")
	}

	_, err := db.NewDelete().Model(o).WherePK().Exec(ctx)
	return err
}

type FloatMatcher struct {
	bun.BaseModel `bun:"table:float_matchers,alias:fm"`

	ID     int64 `bun:",pk,autoincrement"`
	Type   string
	Values []*FloatMatcherElement `bun:"rel:has-many,join:id=matcher_id"`
}

func NewFloatMatcherFromCoRIM(origin *comid.NumberMatcher) (*FloatMatcher, error) {
	var matcher FloatMatcher
	if err := matcher.FromCoRIM(origin); err != nil {
		return nil, err
	}

	return &matcher, nil
}

func SelectFloatMatcher(ctx context.Context, db bun.IDB, id int64) (*FloatMatcher, error) {
	ret := FloatMatcher{ID: id}

	if err := ret.Select(ctx, db); err != nil {
		return nil, err
	}

	return &ret, nil
}

func (o *FloatMatcher) DbID() int64 {
	return o.ID
}

func (o *FloatMatcher) TableName() string {
	return "float_matchers"
}

func (o *FloatMatcher) IsTable() bool {
	return true
}

func (o *FloatMatcher) Insert(ctx context.Context, db bun.IDB) error {
	if _, err := db.NewInsert().Model(o).Exec(ctx); err != nil {
		return err
	}

	for i, value := range o.Values {
		value.MatcherID = o.ID

		if err := value.Insert(ctx, db); err != nil {
			return fmt.Errorf("float matcher value at index %d: %w", i, err)
		}
	}

	return nil
}

func (o *FloatMatcher) Select(ctx context.Context, db bun.IDB) error {
	if o.ID == 0 {
		return errors.New("ID not set")
	}

	err := db.NewSelect().
		Model(o).
		Relation("Values").
		Where("fm.id = ?", o.ID).
		Scan(ctx)

	if err != nil {
		return err
	}

	return nil
}

func (o *FloatMatcher) Delete(ctx context.Context, db bun.IDB) error { // nolint:dupl
	if o.ID == 0 {
		return errors.New("ID not set")
	}

	for i, value := range o.Values {
		if err := value.Delete(ctx, db); err != nil {
			return fmt.Errorf("float matcher value at index %d: %w", i, err)
		}
	}

	_, err := db.NewDelete().Model(o).WherePK().Exec(ctx)
	return err
}

func (o *FloatMatcher) ToCoRIM() (*comid.NumberMatcher, error) {
	numbers := make([]comid.Number, len(o.Values))
	for i, element := range o.Values {
		numbers[i] = comid.NumberFromFloat64(element.Value)
	}

	return matcherFromNumbers(o.Type, numbers)
}

func (o *FloatMatcher) FromCoRIM(origin *comid.NumberMatcher) error {
	var values []*FloatMatcherElement // nolint:prealloc
	for i, originValue := range util.Enumerate(origin.IterValues()) {
		if !originValue.IsFloat64() {
			return fmt.Errorf("float value at index %d", i)
		}

		values = append(values, &FloatMatcherElement{
			Value: originValue.Float64(),
		})
	}

	typ, err := identifyNumberMatcherType(origin)
	if err != nil {
		return err
	}

	o.Type = typ
	o.Values = values

	return nil
}

type FloatMatcherElement struct {
	bun.BaseModel `bun:"table:float_matcher_elements,alias:fme"`

	ID        int64 `bun:",pk,autoincrement"`
	Value     float64
	MatcherID int64
}

func (o *FloatMatcherElement) DbID() int64 {
	return o.ID
}

func (o *FloatMatcherElement) TableName() string {
	return "float_matcher_elements"
}

func (o *FloatMatcherElement) IsTable() bool {
	return true
}

func (o *FloatMatcherElement) Insert(ctx context.Context, db bun.IDB) error {
	_, err := db.NewInsert().Model(o).Exec(ctx)
	return err
}

func (o *FloatMatcherElement) Select(ctx context.Context, db bun.IDB) error {
	if o.ID == 0 {
		return errors.New("ID not set")
	}

	return db.NewSelect().Model(o).Where("fme.id = ?", o.ID).Scan(ctx)
}

func (o *FloatMatcherElement) Delete(ctx context.Context, db bun.IDB) error { // nolint:dupl
	if o.ID == 0 {
		return errors.New("ID not set")
	}

	_, err := db.NewDelete().Model(o).WherePK().Exec(ctx)
	return err
}

type TextMatcher struct {
	bun.BaseModel `bun:"table:text_matchers,alias:tm"`

	ID     int64 `bun:",pk,autoincrement"`
	Type   string
	Values []*TextMatcherElement `bun:"rel:has-many,join:id=matcher_id"`
}

func NewTextMatcherFromCoRIM(origin *comid.TextMatcher) (*TextMatcher, error) {
	var matcher TextMatcher
	if err := matcher.FromCoRIM(origin); err != nil {
		return nil, err
	}

	return &matcher, nil
}

func SelectTextMatcher(ctx context.Context, db bun.IDB, id int64) (*TextMatcher, error) {
	ret := TextMatcher{ID: id}

	if err := ret.Select(ctx, db); err != nil {
		return nil, err
	}

	return &ret, nil
}

func (o *TextMatcher) DbID() int64 {
	return o.ID
}

func (o *TextMatcher) TableName() string {
	return "text_matchers"
}

func (o *TextMatcher) IsTable() bool {
	return true
}

func (o *TextMatcher) Insert(ctx context.Context, db bun.IDB) error {
	if _, err := db.NewInsert().Model(o).Exec(ctx); err != nil {
		return err
	}

	for i, value := range o.Values {
		value.MatcherID = o.ID

		if err := value.Insert(ctx, db); err != nil {
			return fmt.Errorf("text matcher value at index %d: %w", i, err)
		}
	}

	return nil
}

func (o *TextMatcher) Select(ctx context.Context, db bun.IDB) error {
	if o.ID == 0 {
		return errors.New("ID not set")
	}

	err := db.NewSelect().
		Model(o).
		Relation("Values").
		Where("tm.id = ?", o.ID).
		Scan(ctx)

	if err != nil {
		return err
	}

	return nil
}

func (o *TextMatcher) Delete(ctx context.Context, db bun.IDB) error { // nolint:dupl
	if o.ID == 0 {
		return errors.New("ID not set")
	}

	for i, value := range o.Values {
		if err := value.Delete(ctx, db); err != nil {
			return fmt.Errorf("text matcher value at index %d: %w", i, err)
		}
	}

	_, err := db.NewDelete().Model(o).WherePK().Exec(ctx)
	return err
}

func (o *TextMatcher) ToCoRIM() (*comid.TextMatcher, error) {
	if o == nil || len(o.Values) == 0 {
		return nil, errors.New("no values")
	}

	values := make([]string, len(o.Values))
	for i, element := range o.Values {
		values[i] = element.Value
	}

	return comid.NewTextMatcher(values[0], values[1:]...), nil
}

func (o *TextMatcher) FromCoRIM(origin *comid.TextMatcher) error {
	var values []*TextMatcherElement // nolint:prealloc
	for originValue := range origin.IterValues() {
		values = append(values, &TextMatcherElement{
			Value: originValue,
		})
	}

	o.Type = origin.Type().String()
	o.Values = values

	return nil
}

type TextMatcherElement struct {
	bun.BaseModel `bun:"table:text_matcher_elements,alias:tme"`

	ID        int64 `bun:",pk,autoincrement"`
	Value     string
	MatcherID int64
}

func (o *TextMatcherElement) DbID() int64 {
	return o.ID
}

func (o *TextMatcherElement) TableName() string {
	return "text_matcher_elements"
}

func (o *TextMatcherElement) IsTable() bool {
	return true
}

func (o *TextMatcherElement) Insert(ctx context.Context, db bun.IDB) error {
	_, err := db.NewInsert().Model(o).Exec(ctx)
	return err
}

func (o *TextMatcherElement) Select(ctx context.Context, db bun.IDB) error {
	if o.ID == 0 {
		return errors.New("ID not set")
	}

	return db.NewSelect().Model(o).Where("tme.id = ?", o.ID).Scan(ctx)
}

func (o *TextMatcherElement) Delete(ctx context.Context, db bun.IDB) error { // nolint:dupl
	if o.ID == 0 {
		return errors.New("ID not set")
	}

	_, err := db.NewDelete().Model(o).WherePK().Exec(ctx)
	return err
}

type BytesMatcher struct {
	bun.BaseModel `bun:"table:bytes_matchers,alias:bm"`

	ID     int64 `bun:",pk,autoincrement"`
	Type   string
	Values []*BytesMatcherElement `bun:"rel:has-many,join:id=matcher_id"`
}

func NewBytesMatcherFromCoRIM(origin *comid.BytesMatcher) (*BytesMatcher, error) {
	var matcher BytesMatcher
	if err := matcher.FromCoRIM(origin); err != nil {
		return nil, err
	}

	return &matcher, nil
}

func SelectBytesMatcher(ctx context.Context, db bun.IDB, id int64) (*BytesMatcher, error) {
	ret := BytesMatcher{ID: id}

	if err := ret.Select(ctx, db); err != nil {
		return nil, err
	}

	return &ret, nil
}

func (o *BytesMatcher) DbID() int64 {
	return o.ID
}

func (o *BytesMatcher) TableName() string {
	return "bytes_matchers"
}

func (o *BytesMatcher) IsTable() bool {
	return true
}

func (o *BytesMatcher) Insert(ctx context.Context, db bun.IDB) error {
	if _, err := db.NewInsert().Model(o).Exec(ctx); err != nil {
		return err
	}

	for i, value := range o.Values {
		value.MatcherID = o.ID

		if err := value.Insert(ctx, db); err != nil {
			return fmt.Errorf("bytes matcher value at index %d: %w", i, err)
		}
	}

	return nil
}

func (o *BytesMatcher) Select(ctx context.Context, db bun.IDB) error {
	if o.ID == 0 {
		return errors.New("ID not set")
	}

	err := db.NewSelect().
		Model(o).
		Relation("Values").
		Where("bm.id = ?", o.ID).
		Scan(ctx)

	if err != nil {
		return err
	}

	return nil
}

func (o *BytesMatcher) Delete(ctx context.Context, db bun.IDB) error { // nolint:dupl
	if o.ID == 0 {
		return errors.New("ID not set")
	}

	for i, value := range o.Values {
		if err := value.Delete(ctx, db); err != nil {
			return fmt.Errorf("bytes matcher value at index %d: %w", i, err)
		}
	}

	_, err := db.NewDelete().Model(o).WherePK().Exec(ctx)
	return err
}

func (o *BytesMatcher) ToCoRIM() (*comid.BytesMatcher, error) {
	if o == nil || len(o.Values) == 0 {
		return nil, errors.New("no values")
	}

	values := make([][]byte, len(o.Values))
	for i, element := range o.Values {
		values[i] = element.Value
	}

	return comid.NewBytesMatcher(values[0], values[1:]...), nil
}

func (o *BytesMatcher) FromCoRIM(origin *comid.BytesMatcher) error {
	var values []*BytesMatcherElement // nolint:prealloc
	for originValue := range origin.IterValues() {
		values = append(values, &BytesMatcherElement{
			Value: originValue,
		})
	}

	o.Type = origin.Type().String()
	o.Values = values

	return nil
}

type BytesMatcherElement struct {
	bun.BaseModel `bun:"table:bytes_matcher_elements,alias:bme"`

	ID        int64 `bun:",pk,autoincrement"`
	Value     []byte
	MatcherID int64
}

func (o *BytesMatcherElement) DbID() int64 {
	return o.ID
}

func (o *BytesMatcherElement) TableName() string {
	return "bytes_matcher_elements"
}

func (o *BytesMatcherElement) IsTable() bool {
	return true
}

func (o *BytesMatcherElement) Insert(ctx context.Context, db bun.IDB) error {
	_, err := db.NewInsert().Model(o).Exec(ctx)
	return err
}

func (o *BytesMatcherElement) Select(ctx context.Context, db bun.IDB) error {
	if o.ID == 0 {
		return errors.New("ID not set")
	}

	return db.NewSelect().Model(o).Where("bme.id = ?", o.ID).Scan(ctx)
}

func (o *BytesMatcherElement) Delete(ctx context.Context, db bun.IDB) error { // nolint:dupl
	if o.ID == 0 {
		return errors.New("ID not set")
	}

	_, err := db.NewDelete().Model(o).WherePK().Exec(ctx)
	return err
}

func matcherFromNumbers(matcherType string, numbers []comid.Number) (*comid.NumberMatcher, error) {
	switch matcherType {
	case comid.UnitType, comid.SetType:
		if len(numbers) == 0 {
			return nil, errors.New("no values")
		}

		return comid.NewNumberMatcher(numbers[0], numbers[1:]...), nil
	case comid.RangeType:
		if len(numbers) != 2 {
			return nil, fmt.Errorf("expected 2 values for range, got %d", len(numbers))
		}

		return comid.NewNumberRange(numbers[0], numbers[1])
	case "lower-bound":
		if len(numbers) != 1 {
			return nil, fmt.Errorf("expected exactly 1 lower bound, got %d", len(numbers))
		}

		return comid.NewNumberLowerBound(numbers[0]), nil
	case "upper-bound":
		if len(numbers) != 1 {
			return nil, fmt.Errorf("expected exactly 1 upper bound, got %d", len(numbers))
		}

		return comid.NewNumberUpperBound(numbers[0]), nil
	default:
		return nil, fmt.Errorf("unexpected number matcher type %q", matcherType)
	}
}

func identifyNumberMatcherType(matcher *comid.NumberMatcher) (string, error) {
	switch matcher.Type() {
	case comid.MatcherTypeUnit, comid.MatcherTypeSet:
		return matcher.Type().String(), nil
	case comid.MatcherTypeRange:
		intRange := matcher.Variant().(comid.RangeMatcher[comid.Number])
		if intRange.Upper() != nil && intRange.Lower() != nil { // nolint:gocritic
			return comid.RangeType, nil
		} else if intRange.Upper() != nil {
			return "upper-bound", nil
		} else {
			return "lower-bound", nil
		}
	default:
		return "", fmt.Errorf("unexpected int matcher type %q", matcher.Type().String())
	}
}
