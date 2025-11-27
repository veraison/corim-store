package model

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/uptrace/bun"
	"github.com/veraison/corim/comid"
)

const (
	IsConfiguredFlag               int64 = 0
	IsSecureFlag                   int64 = 1
	IsRecoveryFlag                 int64 = 2
	IsDebugFlag                    int64 = 3
	IsReplayProtectedFlag          int64 = 4
	IsIntegrityProtectedFlag       int64 = 5
	IsRuntimeMeasuredFlag          int64 = 6
	IsImmutableFlag                int64 = 7
	IsTcbFlag                      int64 = 8
	IsConfidentialityProtectedFlag int64 = 9
)

func FlagsFromCoRIM(origin *comid.FlagsMap) ([]*Flag, error) {
	var ret []*Flag

	if origin == nil {
		return nil, nil
	}

	if origin.IsConfigured != nil {
		ret = append(ret, NewFlag(IsConfiguredFlag, *origin.IsConfigured))
	}

	if origin.IsSecure != nil {
		ret = append(ret, NewFlag(IsSecureFlag, *origin.IsSecure))
	}

	if origin.IsRecovery != nil {
		ret = append(ret, NewFlag(IsRecoveryFlag, *origin.IsRecovery))
	}

	if origin.IsDebug != nil {
		ret = append(ret, NewFlag(IsDebugFlag, *origin.IsDebug))
	}

	if origin.IsReplayProtected != nil {
		ret = append(ret, NewFlag(IsReplayProtectedFlag, *origin.IsReplayProtected))
	}

	if origin.IsIntegrityProtected != nil {
		ret = append(ret, NewFlag(IsIntegrityProtectedFlag, *origin.IsIntegrityProtected))
	}

	if origin.IsRuntimeMeasured != nil {
		ret = append(ret, NewFlag(IsRuntimeMeasuredFlag, *origin.IsRuntimeMeasured))
	}

	if origin.IsImmutable != nil {
		ret = append(ret, NewFlag(IsImmutableFlag, *origin.IsImmutable))
	}

	if origin.IsTcb != nil {
		ret = append(ret, NewFlag(IsTcbFlag, *origin.IsTcb))
	}

	if origin.IsConfidentialityProtected != nil {
		ret = append(ret, NewFlag(IsConfidentialityProtectedFlag, *origin.IsConfidentialityProtected))
	}

	for _, extVal := range origin.Values() {
		codePoint, err := strconv.Atoi(extVal.CBORTag)
		if err != nil {
			return nil, fmt.Errorf("non-integer CBOR tag: %+v", extVal)
		}

		switch t := extVal.Value.(type) {
		case bool:
			ret = append(ret, NewFlag(int64(codePoint), t))
		case *bool:
			ret = append(ret, NewFlag(int64(codePoint), *t))
		default:
			return nil, fmt.Errorf("invalid Flags extension: %+v", extVal)
		}
	}

	return ret, nil
}

func FlagsToCoRIM(origin []*Flag) (*comid.FlagsMap, error) {
	if len(origin) == 0 {
		return nil, nil
	}

	var ret comid.FlagsMap
	var err error
	extMap := make(map[int64]bool)

	for _, origFlag := range origin {
		switch origFlag.CodePoint {
		case IsConfiguredFlag:
			ret.IsConfigured = &origFlag.Value
		case IsSecureFlag:
			ret.IsSecure = &origFlag.Value
		case IsRecoveryFlag:
			ret.IsRecovery = &origFlag.Value
		case IsDebugFlag:
			ret.IsDebug = &origFlag.Value
		case IsReplayProtectedFlag:
			ret.IsReplayProtected = &origFlag.Value
		case IsIntegrityProtectedFlag:
			ret.IsIntegrityProtected = &origFlag.Value
		case IsRuntimeMeasuredFlag:
			ret.IsRuntimeMeasured = &origFlag.Value
		case IsImmutableFlag:
			ret.IsImmutable = &origFlag.Value
		case IsTcbFlag:
			ret.IsTcb = &origFlag.Value
		case IsConfidentialityProtectedFlag:
			ret.IsConfidentialityProtected = &origFlag.Value
		default:
			extMap[origFlag.CodePoint] = origFlag.Value
		}
	}

	ret.Extensions, err = makeFlagExtensions(extMap)
	if err != nil {
		return nil, err
	}

	return &ret, nil
}

type Flag struct {
	bun.BaseModel `bun:"table:flags,alias:flg"`

	ID int64 `bun:",pk,autoincrement"`

	CodePoint int64
	Value     bool

	MeasurementID int64 `bun:",nullzero"`
}

func NewFlag(cp int64, val bool) *Flag {
	return &Flag{
		CodePoint: cp,
		Value:     val,
	}
}

func (o *Flag) Insert(ctx context.Context, db bun.IDB) error {
	_, err := db.NewInsert().Model(o).Exec(ctx)
	return err
}

func (o *Flag) Select(ctx context.Context, db bun.IDB) error {
	if o.ID == 0 {
		return errors.New("ID not set")
	}

	return db.NewSelect().Model(o).Where("id = ?", o.ID).Scan(ctx)
}

func (o *Flag) Delete(ctx context.Context, db bun.IDB) error {
	if o.ID == 0 {
		return errors.New("ID not set")
	}

	_, err := db.NewDelete().Model(o).WherePK().Exec(ctx)
	return err
}

func makeFlagExtensions(flags map[int64]bool) (comid.Extensions, error) {
	var ret comid.Extensions
	values := make([]bool, 0, len(flags))
	fields := make([]reflect.StructField, 0, len(flags))

	if len(flags) == 0 {
		return ret, nil
	}

	for key, val := range flags {
		fields = append(fields, reflect.StructField{
			Name: strings.ReplaceAll(fmt.Sprintf("Flag%d", key), "-", "_"),
			Type: reflect.TypeOf(val),
			Tag:  reflect.StructTag(fmt.Sprintf("cbor:\"%d,keyasint\" json:\"flag%d\"", key, key)),
		})
		values = append(values, val)
	}

	structType := reflect.StructOf(fields)
	structPtr := reflect.New(structType)
	structValue := structPtr.Elem()

	for i, val := range values {
		field := structValue.FieldByIndex([]int{i})
		if field.IsValid() && field.CanSet() {
			field.Set(reflect.ValueOf(val))
		} else {
			return comid.Extensions{}, fmt.Errorf("could not set field %s", field.Type().Name())
		}
	}

	ret.IMapValue = structPtr.Interface()

	return ret, nil
}
