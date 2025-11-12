package model

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/fxamacker/cbor/v2"
	"github.com/uptrace/bun"
	"github.com/veraison/corim/comid"
	"github.com/veraison/eat"
	"github.com/veraison/swid"
)

const (
	MvalVersion            int64 = 0
	MvalSvn                int64 = 1
	MvalDigests            int64 = 2
	MvalFlags              int64 = 3
	MvalRawValue           int64 = 4
	MvalMACAddr            int64 = 6
	MvalIPAddr             int64 = 7
	MvalSerialNumber       int64 = 8
	MvalUEID               int64 = 9
	MvalUUID               int64 = 10
	MvalName               int64 = 11
	MvalCryptoKeys         int64 = 13
	MvalIntegrityRegisters int64 = 14
	MvalIntRange           int64 = 15
)

type MeasurementValueEntry struct {
	bun.BaseModel `bun:"table:measurement_value_entries,alias:mve"`

	ID int64 `bun:",pk,autoincrement"`

	CodePoint  int64
	ValueType  string
	ValueBytes *[]byte
	ValueText  *string
	ValueInt   *int64

	MeasurementID int64
}

func (o *MeasurementValueEntry) Insert(ctx context.Context, db bun.IDB) error {
	_, err := db.NewInsert().Model(o).Exec(ctx)
	return err
}

func (o *MeasurementValueEntry) Delete(ctx context.Context, db bun.IDB) error {
	if o.ID == 0 {
		return errors.New("ID not set")
	}

	_, err := db.NewDelete().Model(o).WherePK().Exec(ctx)
	return err
}

func MeasurementsFromCoRIM(origin comid.Measurements) ([]*Measurement, error) {
	if origin.IsEmpty() {
		return nil, nil
	}

	ret := make([]*Measurement, 0, len(origin.Values))

	for i, originMea := range origin.Values {
		mea, err := NewMeasurementFromCoRIM(&originMea)
		if err != nil {
			return nil, fmt.Errorf("could not construct measurement at index %d: %w", i, err)
		}

		ret = append(ret, mea)
	}

	return ret, nil
}

func MeasurementsToCoRIM(origin []*Measurement) (comid.Measurements, error) {
	ret := comid.NewMeasurements()

	for i, originMea := range origin {
		corimMea, err := originMea.ToCoRIM()
		if err != nil {
			return comid.Measurements{}, fmt.Errorf(
				"could not conver measurement at index %d: %w", i, err)
		}

		ret.Add(corimMea)
	}

	return *ret, nil
}

type Measurement struct {
	bun.BaseModel `bun:"table:measurements,alias:mea"`

	ID int64 `bun:",pk,autoincrement"`

	KeyType  *string
	KeyBytes *[]byte

	ValueEntries       []*MeasurementValueEntry `bun:"rel:has-many,join:id=measurement_id"`
	Digests            []*Digest                `bun:"rel:has-many,join:id=owner_id,join:type=owner_type,polymorphic:measurement"`
	Flags              []*Flag                  `bun:"rel:has-many,join:id=measurement_id"`
	IntegrityRegisters []*IntegrityRegister     `bun:"rel:has-many,join:id=measurement_id"`
	Extensions         []*ExtensionValue        `bun:"rel:has-many,join:id=owner_id,join:type=owner_type,polymorphic:measurement"`

	AuthorizedBy []*CryptoKey `bun:"rel:has-many,join:id=owner_id,join:type=owner_type,polymorphic:measurement"`
	OwnerID      int64        `bun:",nullzero"`
	OwnerType    string       `bun:",nullzero"`
}

func NewMeasurementFromCoRIM(origin *comid.Measurement) (*Measurement, error) {
	var ret Measurement

	err := ret.FromCoRIM(origin)
	if err != nil {
		return nil, err
	}

	return &ret, nil
}

func SelectMeasurement(ctx context.Context, db bun.IDB, id int64) (*Measurement, error) {
	ret := Measurement{ID: id}

	if err := ret.Select(ctx, db); err != nil {
		return nil, err
	}

	return &ret, nil
}

func (o *Measurement) FromCoRIM(origin *comid.Measurement) error {
	if origin.Key != nil {
		mkeyType := origin.Key.Type()
		var mkeyBytes []byte

		switch t := origin.Key.Value.(type) {
		case comid.UintMkey:
			mkeyBytes = make([]byte, 8)
			binary.BigEndian.PutUint64(mkeyBytes, uint64(t))
		case comid.StringMkey:
			mkeyBytes = []byte(t)
		case comid.TaggedUUID:
			mkeyBytes = t[:]
		case *comid.TaggedUUID:
			mkeyBytes = (*t)[:]
		case *comid.TaggedOID:
			mkeyBytes = (*t)[:]
		default:
			var err error
			mkeyBytes, err = origin.Key.MarshalCBOR()
			if err != nil {
				return fmt.Errorf("could not CBOR-encode group: %w", err)
			}
		}

		o.KeyType = &mkeyType
		o.KeyBytes = &mkeyBytes
	}

	if origin.Val.Ver != nil {
		o.ValueEntries = append(o.ValueEntries, &MeasurementValueEntry{
			CodePoint: MvalVersion,
			ValueType: origin.Val.Ver.Scheme.String(),
			ValueText: &origin.Val.Ver.Version,
		})
	}

	if origin.Val.SVN != nil {
		var svn int64

		switch t := origin.Val.SVN.Value.(type) {
		case comid.TaggedSVN:
			svn = int64(t)
		case *comid.TaggedSVN:
			svn = int64(*t)
		case comid.TaggedMinSVN:
			svn = int64(t)
		case *comid.TaggedMinSVN:
			svn = int64(*t)
		default:
			return fmt.Errorf("unexpected SVN type: %T", t)
		}

		o.ValueEntries = append(o.ValueEntries, &MeasurementValueEntry{
			CodePoint: MvalSvn,
			ValueType: origin.Val.SVN.Value.Type(),
			ValueInt:  &svn,
		})
	}

	digests, err := DigestsFromCoRIM(origin.Val.Digests)
	if err != nil {
		return err
	}
	o.Digests = digests

	flags, err := FlagsFromCoRIM(origin.Val.Flags)
	if err != nil {
		return err
	}
	o.Flags = flags

	if origin.Val.RawValue != nil {
		bytes, err := origin.Val.RawValue.GetBytes()
		if err != nil {
			return fmt.Errorf("could not get RawValue bytes: %w", err)
		}

		o.ValueEntries = append(o.ValueEntries, &MeasurementValueEntry{
			CodePoint:  MvalRawValue,
			ValueType:  "bytes",
			ValueBytes: &bytes,
		})
	}

	if origin.Val.MACAddr != nil {
		o.ValueEntries = append(o.ValueEntries, &MeasurementValueEntry{
			CodePoint:  MvalMACAddr,
			ValueType:  "bytes",
			ValueBytes: (*[]byte)(origin.Val.MACAddr),
		})
	}

	if origin.Val.IPAddr != nil {
		o.ValueEntries = append(o.ValueEntries, &MeasurementValueEntry{
			CodePoint:  MvalIPAddr,
			ValueType:  "bytes",
			ValueBytes: (*[]byte)(origin.Val.IPAddr),
		})
	}

	if origin.Val.SerialNumber != nil {
		o.ValueEntries = append(o.ValueEntries, &MeasurementValueEntry{
			CodePoint: MvalSerialNumber,
			ValueType: "string",
			ValueText: origin.Val.SerialNumber,
		})
	}

	if origin.Val.UEID != nil {
		o.ValueEntries = append(o.ValueEntries, &MeasurementValueEntry{
			CodePoint:  MvalUEID,
			ValueType:  "bytes",
			ValueBytes: (*[]byte)(origin.Val.UEID),
		})
	}

	if origin.Val.UUID != nil {
		bytes := (*origin.Val.UUID)[:]
		o.ValueEntries = append(o.ValueEntries, &MeasurementValueEntry{
			CodePoint:  MvalUUID,
			ValueType:  "bytes",
			ValueBytes: &bytes,
		})
	}

	if origin.Val.Name != nil {
		o.ValueEntries = append(o.ValueEntries, &MeasurementValueEntry{
			CodePoint: MvalName,
			ValueType: "string",
			ValueText: origin.Val.Name,
		})
	}

	integRegs, err := IntegerityRegistersFromCoRIM(origin.Val.IntegrityRegisters)
	if err != nil {
		return err
	}
	o.IntegrityRegisters = integRegs

	exts, err := CoMIDExtensionsFromCoRIM(origin.Val.Extensions)
	if err != nil {
		return err
	}
	o.Extensions = exts

	o.AuthorizedBy, err = CryptoKeysFromCoRIM(origin.AuthorizedBy)
	if err != nil {
		return err
	}

	return nil
}

func (o *Measurement) ToCoRIM() (*comid.Measurement, error) {
	var err error
	var mkey *comid.Mkey = nil
	var mval comid.Mval

	if o.KeyType != nil {
		if o.KeyBytes == nil {
			return nil, errors.New("missing mkey data")
		}

		switch *o.KeyType {
		case comid.OIDType:
			mkey, err = comid.NewMkeyOID(*o.KeyBytes)
			if err != nil {
				return nil, fmt.Errorf("could not initialize OID mkey: %w", err)
			}
		case comid.UUIDType:
			mkey, err = comid.NewMkeyUUID(*o.KeyBytes)
			if err != nil {
				return nil, fmt.Errorf("could not initialize UUID mkey: %w", err)
			}
		case comid.UintType:
			var val uint64
			reader := bytes.NewReader(*o.KeyBytes)
			err = binary.Read(reader, binary.BigEndian, &val)
			if err != nil {
				return nil, fmt.Errorf("could not parse uint64: %w", err)
			}

			mkey, err = comid.NewMkeyUint(val)
			if err != nil {
				return nil, fmt.Errorf("could not create UintMkey: %w", err)
			}
		case comid.StringType:
			mkey, err = comid.NewMkeyString(*o.KeyBytes)
			if err != nil {
				return nil, fmt.Errorf("could not initialize string mkey: %w", err)
			}
		default:
			mkey, err = comid.NewMkey(nil, *o.KeyType)
			if err != nil {
				return nil, err
			}

			if err = cbor.Unmarshal(*o.KeyBytes, &mkey.Value); err != nil {
				return nil, fmt.Errorf("could not CBOR-decode mkey: %w", err)
			}
		}
	}

	for _, entry := range o.ValueEntries {
		switch entry.CodePoint {
		case MvalVersion:
			if entry.ValueText == nil {
				return nil, fmt.Errorf("missing version data: %+v", entry)
			}

			scheme, err := parseVersionScheme(entry.ValueType)
			if err != nil {
				return nil, err
			}

			version := comid.Version{
				Version: *entry.ValueText,
				Scheme:  scheme,
			}

			mval.Ver = &version
		case MvalSvn:
			if entry.ValueInt == nil {
				return nil, fmt.Errorf("missing SVN data: %+v", entry)
			}

			switch entry.ValueType {
			case comid.MinValueType:
				mval.SVN, err = comid.NewTaggedMinSVN(*entry.ValueInt)
				if err != nil {
					return nil, fmt.Errorf("could not create TaggedMinSVN: %w", err)
				}
			case comid.ExactValueType:
				mval.SVN, err = comid.NewTaggedSVN(*entry.ValueInt)
				if err != nil {
					return nil, fmt.Errorf("could not create TaggedSVN: %w", err)
				}
			default:
				return nil, fmt.Errorf("unexpected SVN type: %s", entry.ValueType)
			}
		case MvalRawValue:
			if entry.ValueBytes == nil {
				return nil, fmt.Errorf("missing RawValue data: %+v", entry)
			}

			switch entry.ValueType {
			case "bytes":
				var val comid.RawValue
				val.SetBytes(*entry.ValueBytes)
				mval.RawValue = &val
			default:
				return nil, fmt.Errorf("unexpected RawValue type: %s", entry.ValueType)
			}
		case MvalMACAddr:
			if entry.ValueBytes == nil {
				return nil, fmt.Errorf("missing MACAddr data: %+v", entry)
			}

			switch entry.ValueType {
			case "bytes":
				mval.MACAddr = (*comid.MACaddr)(entry.ValueBytes)
			default:
				return nil, fmt.Errorf("unexpected MACAddr type: %s", entry.ValueType)
			}
		case MvalIPAddr:
			if entry.ValueBytes == nil {
				return nil, fmt.Errorf("missing IPAddr data: %+v", entry)
			}

			switch entry.ValueType {
			case "bytes":
				mval.IPAddr = (*net.IP)(entry.ValueBytes)
			default:
				return nil, fmt.Errorf("unexpected IPAddr type: %s", entry.ValueType)
			}
		case MvalSerialNumber:
			if entry.ValueText == nil {
				return nil, fmt.Errorf("missing SerialNumber data: %+v", entry)
			}

			switch entry.ValueType {
			case "string":
				mval.SerialNumber = entry.ValueText
			default:
				return nil, fmt.Errorf("unexpected SerialNumber type: %s", entry.ValueType)
			}
		case MvalUEID:
			if entry.ValueBytes == nil {
				return nil, fmt.Errorf("missing UEID data: %+v", entry)
			}

			switch entry.ValueType {
			case "bytes":
				mval.UEID = (*eat.UEID)(entry.ValueBytes)
			default:
				return nil, fmt.Errorf("unexpected UEID type: %s", entry.ValueType)
			}
		case MvalUUID:
			if entry.ValueBytes == nil {
				return nil, fmt.Errorf("missing UUID data: %+v", entry)
			}

			switch entry.ValueType {
			case "bytes":
				val, err := comid.NewTaggedUUID(*entry.ValueBytes)
				if err != nil {
					return nil, fmt.Errorf("could not construct UUID: %w", err)
				}

				mval.UUID = (*comid.UUID)(val)
			default:
				return nil, fmt.Errorf("unexpected UUID type: %s", entry.ValueType)
			}
		case MvalName:
			if entry.ValueText == nil {
				return nil, fmt.Errorf("missing Name data: %+v", entry)
			}

			switch entry.ValueType {
			case "string":
				mval.Name = entry.ValueText
			default:
				return nil, fmt.Errorf("unexpected Name type: %s", entry.ValueType)
			}
		case MvalDigests, MvalFlags:
			return nil, fmt.Errorf(
				"unexpected value entry for code point %d (should be in its own table)",
				entry.CodePoint,
			)
		default:
			return nil, fmt.Errorf("unexpted code point: %d", entry.CodePoint)
		}
	}

	mval.Digests, err = DigestsToCoRIM(o.Digests)
	if err != nil {
		return nil, err
	}

	mval.Flags, err = FlagsToCoRIM(o.Flags)
	if err != nil {
		return nil, err
	}

	mval.IntegrityRegisters, err = IntegerityRegistersToCoRIM(o.IntegrityRegisters)
	if err != nil {
		return nil, err
	}

	mval.Extensions, err = CoMIDExtensionsToCoRIM(o.Extensions)
	if err != nil {
		return nil, err
	}

	auth, err := CryptoKeysToCoRIM(o.AuthorizedBy)
	if err != nil {
		return nil, err
	}

	ret := comid.Measurement{
		Key:          mkey,
		Val:          mval,
		AuthorizedBy: auth,
	}

	return &ret, nil
}

func (o *Measurement) Insert(ctx context.Context, db bun.IDB) error {
	if _, err := db.NewInsert().Model(o).Exec(ctx); err != nil {
		return err
	}

	for _, entry := range o.ValueEntries {
		entry.MeasurementID = o.ID

		if err := entry.Insert(ctx, db); err != nil {
			return fmt.Errorf("error inserting digest %+v: %w", entry, err)
		}
	}

	for i, digest := range o.Digests {
		digest.OwnerID = o.ID
		digest.OwnerType = "measurement"

		if err := digest.Insert(ctx, db); err != nil {
			return fmt.Errorf("error inserting digest %d: %w", i, err)
		}
	}

	for _, flag := range o.Flags {
		flag.MeasurementID = o.ID

		if err := flag.Insert(ctx, db); err != nil {
			return fmt.Errorf("error inserting flag %d: %w", flag.CodePoint, err)
		}
	}

	for i, reg := range o.IntegrityRegisters {
		reg.MeasurementID = o.ID

		if err := reg.Insert(ctx, db); err != nil {
			return fmt.Errorf("error inserting integerity register at index %d: %w", i, err)
		}
	}

	for _, ext := range o.Extensions {
		ext.OwnerID = o.ID
		ext.OwnerType = "measurement"

		if err := ext.Insert(ctx, db); err != nil {
			return fmt.Errorf("error inserting extension %+v: %w", ext, err)
		}
	}

	for _, key := range o.AuthorizedBy {
		key.OwnerID = o.ID
		key.OwnerType = "measurement"

		if err := key.Insert(ctx, db); err != nil {
			return fmt.Errorf("error inserting crypto key %+v: %w", key, err)
		}
	}

	return nil
}

func (o *Measurement) Select(ctx context.Context, db bun.IDB) error {
	if o.ID == 0 {
		return errors.New("ID not set")
	}

	err := db.NewSelect().
		Model(o).
		Relation("AuthorizedBy").
		Relation("Digests").
		Relation("Flags").
		Relation("IntegrityRegisters.Digests").
		Relation("ValueEntries").
		Relation("Extensions").
		Where("mea.id = ?", o.ID).
		Scan(ctx)

	if err != nil {
		return err
	}

	return nil
}

func (o *Measurement) Delete(ctx context.Context, db bun.IDB) error { // nolint:dupl
	if o.ID == 0 {
		return errors.New("ID not set")
	}

	for i, key := range o.AuthorizedBy {
		if err := key.Delete(ctx, db); err != nil {
			return fmt.Errorf("authorized-by key at index %d: %w", i, err)
		}
	}

	for i, digest := range o.Digests {
		if err := digest.Delete(ctx, db); err != nil {
			return fmt.Errorf("digest at index %d: %w", i, err)
		}
	}

	for i, flag := range o.Flags {
		if err := flag.Delete(ctx, db); err != nil {
			return fmt.Errorf("flag at index %d: %w", i, err)
		}
	}

	for i, register := range o.IntegrityRegisters {
		if err := register.Delete(ctx, db); err != nil {
			return fmt.Errorf("integrity register at index %d: %w", i, err)
		}
	}

	for i, entry := range o.ValueEntries {
		if err := entry.Delete(ctx, db); err != nil {
			return fmt.Errorf("value entry at index %d: %w", i, err)
		}
	}

	for i, extension := range o.Extensions {
		if err := extension.Delete(ctx, db); err != nil {
			return fmt.Errorf("extension at index %d: %w", i, err)
		}
	}

	_, err := db.NewDelete().Model(o).WherePK().Exec(ctx)
	return err
}

func parseVersionScheme(text string) (swid.VersionScheme, error) {
	var ret swid.VersionScheme

	switch text {
	case "multipartnumeric":
		if err := ret.SetCode(swid.VersionSchemeMultipartNumeric); err != nil {
			return ret, err
		}
	case "multipartnumeric+suffix":
		if err := ret.SetCode(swid.VersionSchemeMultipartNumericSuffix); err != nil {
			return ret, err
		}
	case "alphanumeric":
		if err := ret.SetCode(swid.VersionSchemeAlphaNumeric); err != nil {
			return ret, err
		}
	case "decimal":
		if err := ret.SetCode(swid.VersionSchemeDecimal); err != nil {
			return ret, err
		}
	case "semver":
		if err := ret.SetCode(swid.VersionSchemeSemVer); err != nil {
			return ret, err
		}
	default:
		if strings.HasPrefix(text, "version-scheme(") {
			code, err := strconv.Atoi(text[15 : len(text)-1])
			if err != nil {
				return ret, fmt.Errorf("could not parse version scheme %q: %w", text, err)
			}
			if err := ret.SetCode(int64(code)); err != nil {
				return ret, err
			}
		} else {
			return ret, fmt.Errorf("invalid version scheme: %s", text)
		}
	}

	return ret, nil
}
