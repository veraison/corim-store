package model

import (
	"context"
	"errors"
	"fmt"
	"unicode/utf8"

	"github.com/fxamacker/cbor/v2"
	"github.com/uptrace/bun"
	"github.com/veraison/corim/comid"
)

func CryptoKeysFromCoRIM(origin *comid.CryptoKeys) ([]*CryptoKey, error) {
	if origin == nil {
		return nil, nil
	}

	ret := make([]*CryptoKey, 0, len(*origin))

	for i, originKey := range *origin {
		ckey, err := NewCryptoKeyFromCoRIM(originKey)
		if err != nil {
			return nil, fmt.Errorf("could not construct crypto key at index %d: %w", i, err)
		}

		ret = append(ret, ckey)
	}

	return ret, nil
}

func CryptoKeysToCoRIM(origin []*CryptoKey) (*comid.CryptoKeys, error) {
	if len(origin) == 0 {
		return nil, nil
	}

	var ret comid.CryptoKeys

	for i, originKey := range origin {
		comidKey, err := originKey.ToCoRIM()
		if err != nil {
			return nil, fmt.Errorf("could not create comid.CryptoKey at index %d: %w", i, err)
		}

		ret = append(ret, comidKey)
	}

	return &ret, nil
}

type CryptoKey struct {
	bun.BaseModel `bun:"table:cryptokeys,alias:ck"`

	ID int64 `bun:",pk,autoincrement"`

	KeyType  string
	KeyBytes []byte

	OwnerID   int64  `bun:",nullzero"`
	OwnerType string `bun:",nullzero"`
}

func NewCryptoKeyFromCoRIM(origin *comid.CryptoKey) (*CryptoKey, error) {
	var ret CryptoKey

	if err := ret.FromCoRIM(origin); err != nil {
		return nil, err
	}

	return &ret, nil
}

func SelectCryptoKey(ctx context.Context, db bun.IDB, id int64) (*CryptoKey, error) {
	var ret CryptoKey

	if err := db.NewSelect().Model(&ret).Where("id = ?", id).Scan(ctx); err != nil {
		return nil, err
	}

	return &ret, nil
}

func (o *CryptoKey) FromCoRIM(origin *comid.CryptoKey) error {
	if origin == nil {
		return errors.New("nil origin")
	}

	var keyBytes []byte
	var err error
	keyType := origin.Type()

	switch keyType {
	case comid.PKIXBase64KeyType, comid.PKIXBase64CertType, comid.PKIXBase64CertPathType,
		comid.ThumbprintType, comid.CertThumbprintType, comid.CertPathThumbprintType:
		keyBytes = []byte(origin.String())
	case comid.BytesType, comid.COSEKeyType:
		switch t := origin.Value.(type) {
		case comid.TaggedBytes:
			keyBytes = []byte(t)
		case *comid.TaggedBytes:
			keyBytes = []byte(*t)
		case comid.TaggedCOSEKey:
			keyBytes = []byte(t)
		case *comid.TaggedCOSEKey:
			keyBytes = []byte(*t)
		default:
			return fmt.Errorf("expected crypto key bytes, found: %T", t)
		}
	default:
		keyBytes, err = origin.MarshalCBOR()
		if err != nil {
			return fmt.Errorf("could not CBOR-encode crypto key: %w", err)
		}
	}

	o.KeyType = keyType
	o.KeyBytes = keyBytes

	return nil
}

func (o *CryptoKey) ToCoRIM() (*comid.CryptoKey, error) {
	switch o.KeyType {
	case comid.PKIXBase64KeyType, comid.PKIXBase64CertType, comid.PKIXBase64CertPathType,
		comid.ThumbprintType, comid.CertThumbprintType, comid.CertPathThumbprintType:
		if !utf8.Valid(o.KeyBytes) {
			return nil, fmt.Errorf("data for %s must be a valid UTF-8 string", o.KeyType)
		}
		return cryptoKeyFactory[o.KeyType](string(o.KeyBytes))
	case comid.BytesType:
		return comid.NewCryptoKeyTaggedBytes(o.KeyBytes)
	case comid.COSEKeyType:
		return comid.NewCOSEKey(o.KeyBytes)
	default:
		ret, err := comid.NewCryptoKey(nil, o.KeyType)
		if err != nil {
			return nil, err
		}

		if err = cbor.Unmarshal(o.KeyBytes, ret); err != nil {
			return nil, fmt.Errorf("could not CBOR-decode crypto key: %w", err)
		}

		return ret, nil
	}
}

func (o *CryptoKey) Insert(ctx context.Context, db bun.IDB) error {
	_, err := db.NewInsert().Model(o).Exec(ctx)
	return err
}

func (o *CryptoKey) Select(ctx context.Context, db bun.IDB) error {
	if o.ID == 0 {
		return errors.New("ID not set")
	}

	_, err := db.NewSelect().Model(o).Where("ck.id = ?", o.ID).Exec(ctx)
	return err
}

func (o *CryptoKey) Delete(ctx context.Context, db bun.IDB) error {
	if o.ID == 0 {
		return errors.New("ID not set")
	}

	_, err := db.NewDelete().Model(o).WherePK().Exec(ctx)
	return err
}

// This is a sub-set of crypto key types that are constructable from string representations.
var cryptoKeyFactory = map[string]comid.ICryptoKeyFactory{
	comid.PKIXBase64KeyType:      comid.NewPKIXBase64Key,
	comid.PKIXBase64CertType:     comid.NewPKIXBase64Cert,
	comid.PKIXBase64CertPathType: comid.NewPKIXBase64CertPath,
	comid.ThumbprintType:         comid.NewThumbprint,
	comid.CertThumbprintType:     comid.NewCertThumbprint,
	comid.CertPathThumbprintType: comid.NewCertPathThumbprint,
}
