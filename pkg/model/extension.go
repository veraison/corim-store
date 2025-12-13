package model

import (
	"context"
	"errors"
	"fmt"
	"reflect"

	"github.com/fxamacker/cbor/v2"
	"github.com/uptrace/bun"
	"github.com/veraison/corim/comid"
	"github.com/veraison/corim/corim"
)

func CoRIMExtensionsFromCoRIM(origin corim.Extensions) ([]*ExtensionValue, error) {
	return CoMIDExtensionsFromCoRIM(comid.Extensions(origin))
}

func CoMIDExtensionsFromCoRIM(origin comid.Extensions) ([]*ExtensionValue, error) {
	var ret []*ExtensionValue // nolint: prealloc
	if origin.IsEmpty() {
		return ret, nil
	}

	extType := reflect.TypeOf(origin.IMapValue)
	extVal := reflect.ValueOf(origin.IMapValue)
	if extType.Kind() == reflect.Pointer {
		extType = extType.Elem()
		extVal = extVal.Elem()
	}

	for i := 0; i < extVal.NumField(); i++ {
		typeField := extType.Field(i)

		fieldJSONTag, _ := typeField.Tag.Lookup("json")
		fieldCBORTag, _ := typeField.Tag.Lookup("cbor")

		retVal := ExtensionValue{
			FieldKind: typeField.Type.Kind(),
			FieldName: typeField.Name,
			JSONTag:   fieldJSONTag,
			CBORTag:   fieldCBORTag,
		}

		extValField := extVal.Field(i)
		// if the value is a pointer, dereference it in cases it points
		// to a base type we don't have to CBOR-encode (e.g. a
		// *string).
		if retVal.FieldKind == reflect.Pointer && !extValField.IsNil() {
			extValField = extValField.Elem()
			retVal.FieldKind = extValField.Kind()
		}

		var err error

		switch retVal.FieldKind {
		case reflect.String:
			retVal.ValueText = extValField.String()
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			retVal.ValueInt = extValField.Int()
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			retVal.ValueInt = int64(extValField.Uint())
		case reflect.Float32, reflect.Float64:
			retVal.ValueFloat = extValField.Float()
		case reflect.Bool:
			if extValField.Bool() {
				retVal.ValueInt = 1
			} else {
				retVal.ValueInt = 0
			}
		default:
			retVal.ValueBytes, err = cbor.Marshal(extValField.Interface())
			if err != nil {
				return nil, fmt.Errorf("error CBOR encoding %s: %w", retVal.FieldName, err)
			}
		}

		ret = append(ret, &retVal)
	}

	for k, v := range origin.Cached {
		bytes, err := cbor.Marshal(v)
		if err != nil {
			return nil, fmt.Errorf("error CBOR encoding cached extension %s: %w", k, err)
		}

		retVal := ExtensionValue{
			FieldName:  "", // empty field name indicates cached value
			JSONTag:    k,
			ValueBytes: bytes,
		}

		ret = append(ret, &retVal)
	}

	return ret, nil
}

func CoRIMExtensionsToCoRIM(origin []*ExtensionValue) (corim.Extensions, error) {
	ret, err := CoMIDExtensionsToCoRIM(origin)
	return corim.Extensions(ret), err
}

func CoMIDExtensionsToCoRIM(origin []*ExtensionValue) (comid.Extensions, error) {
	if len(origin) == 0 {
		return comid.Extensions{}, nil
	}

	values := make(map[string]any, len(origin))
	fields := make([]reflect.StructField, 0, len(origin))
	cached := make(map[string]any, len(origin))

	for _, origVal := range origin {
		var val any

		if origVal.FieldName == "" {
			// empty field name means this is a cached value
			if err := cbor.Unmarshal(origVal.ValueBytes, &val); err != nil {
				return comid.Extensions{}, fmt.Errorf(
					"error decoding CBOR for %s: %w", origVal.JSONTag, err)
			}

			cached[origVal.JSONTag] = val
			continue
		}

		switch origVal.FieldKind {
		case reflect.String:
			val = origVal.ValueText
		case reflect.Int:
			val = int(origVal.ValueInt)
		case reflect.Int8:
			val = int8(origVal.ValueInt)
		case reflect.Int16:
			val = int16(origVal.ValueInt)
		case reflect.Int32:
			val = int32(origVal.ValueInt)
		case reflect.Int64:
			val = int64(origVal.ValueInt)
		case reflect.Uint:
			val = uint(origVal.ValueInt)
		case reflect.Uint8:
			val = uint8(origVal.ValueInt)
		case reflect.Uint16:
			val = uint16(origVal.ValueInt)
		case reflect.Uint32:
			val = uint32(origVal.ValueInt)
		case reflect.Uint64:
			val = uint64(origVal.ValueInt)
		case reflect.Float32:
			val = float32(origVal.ValueFloat)
		case reflect.Float64:
			val = origVal.ValueFloat
		case reflect.Bool:
			if origVal.ValueInt == 0 {
				val = false
			} else {
				val = true
			}
		default:
			if len(origVal.ValueBytes) != 0 {
				err := cbor.Unmarshal(origVal.ValueBytes, &val)
				if err != nil {
					return comid.Extensions{}, fmt.Errorf(
						"error CBOR decoding %s: %w", origVal.FieldName, err)
				}
			}
		}

		fields = append(fields, reflect.StructField{
			Name: origVal.FieldName,
			Type: reflect.TypeOf(val),
			Tag: reflect.StructTag(fmt.Sprintf("cbor:\"%s\" json:\"%s\"",
				origVal.CBORTag, origVal.JSONTag)),
		})

		values[origVal.FieldName] = val
	}

	structType := reflect.StructOf(fields)
	structPtr := reflect.New(structType)
	structValue := structPtr.Elem()

	for _, origVal := range origin {
		if origVal.FieldName == "" {
			continue
		}

		field := structValue.FieldByName(origVal.FieldName)
		if field.IsValid() && field.CanSet() {
			field.Set(reflect.ValueOf(values[origVal.FieldName]))
		} else {
			return comid.Extensions{}, fmt.Errorf("could not set field %q", origVal.FieldName)
		}
	}

	var ret comid.Extensions
	ret.IMapValue = structPtr.Interface()

	if len(cached) != 0 {
		ret.Cached = cached
	}

	return ret, nil
}

type ExtensionValue struct {
	bun.BaseModel `bun:"table:extensions,alias:ext"`

	ID int64 `bun:",pk,autoincrement"`

	FieldKind reflect.Kind
	FieldName string
	JSONTag   string
	CBORTag   string

	ValueBytes []byte
	ValueText  string
	ValueInt   int64
	ValueFloat float64

	OwnerID   int64  `bun:",nullzero"`
	OwnerType string `bun:",nullzero"`
}

func (o *ExtensionValue) Insert(ctx context.Context, db bun.IDB) error {
	_, err := db.NewInsert().Model(o).Exec(ctx)
	return err
}

func (o *ExtensionValue) Select(ctx context.Context, db bun.IDB) error {
	if o.ID == 0 {
		return errors.New("ID not set")
	}

	return db.NewSelect().Model(o).Where("id = ?", o.ID).Scan(ctx)
}

func (o *ExtensionValue) Delete(ctx context.Context, db bun.IDB) error {
	if o.ID == 0 {
		return errors.New("ID not set")
	}

	_, err := db.NewDelete().Model(o).WherePK().Exec(ctx)
	return err
}
