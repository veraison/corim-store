package model

import (
	"context"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/fxamacker/cbor/v2"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
	"github.com/veraison/corim/comid"
)

type Environment struct {
	bun.BaseModel `bun:"table:environments,alias:env"`

	ID int64 `bun:",pk,autoincrement"`

	// note: since there is a 1-to-1 correspondence between Environment and Class,
	// and Class is not used anywhere else in CoRIM, Class is "collapsed" into the
	// Environment in order to simplify the schema and avoid a needless join.

	ClassType  *string
	ClassBytes *[]byte
	Vendor     *string
	Model      *string
	Layer      *uint64
	Index      *uint64

	InstanceType  *string
	InstanceBytes *[]byte

	GroupType  *string
	GroupBytes *[]byte
}

func NewEnvironmentFromCoRIM(origin *comid.Environment) (*Environment, error) {
	var ret Environment

	if err := ret.FromCoRIM(origin); err != nil {
		return nil, err
	}

	return &ret, nil
}

func SelectEnvironment(ctx context.Context, db bun.IDB, id int64) (*Environment, error) {
	var ret Environment

	if err := db.NewSelect().Model(&ret).Where("id = ?", id).Scan(ctx); err != nil {
		return nil, err
	}

	return &ret, nil
}

func (o *Environment) FromCoRIM(origin *comid.Environment) error {
	var err error

	if origin.Class != nil {
		if origin.Class.ClassID != nil {
			classType := origin.Class.ClassID.Type()
			var classBytes []byte

			switch classType {
			case comid.OIDType, comid.UUIDType, comid.BytesType:
				classBytes = origin.Class.ClassID.Bytes()
			default:
				classBytes, err = origin.Class.ClassID.MarshalCBOR()
				if err != nil {
					return fmt.Errorf("could not CBOR-encode class ID: %w", err)
				}
			}

			o.ClassType = &classType
			o.ClassBytes = &classBytes
		}

		o.Vendor = origin.Class.Vendor
		o.Model = origin.Class.Model
		o.Layer = origin.Class.Layer
		o.Index = origin.Class.Index
	}

	if origin.Instance != nil {
		instanceType := origin.Instance.Type()
		var instanceBytes []byte

		switch instanceType {
		case "ueid", comid.UUIDType, comid.BytesType:
			instanceBytes = origin.Instance.Bytes()
		default:
			instanceBytes, err = origin.Instance.MarshalCBOR()
			if err != nil {
				return fmt.Errorf("could not CBOR-encode instance: %w", err)
			}
		}

		o.InstanceType = &instanceType
		o.InstanceBytes = &instanceBytes
	}

	if origin.Group != nil {
		groupType := origin.Group.Type()
		var groupBytes []byte

		switch groupType {
		case comid.UUIDType, comid.BytesType:
			groupBytes = origin.Group.Bytes()
		default:
			groupBytes, err = origin.Group.MarshalCBOR()
			if err != nil {
				return fmt.Errorf("could not CBOR-encode group: %w", err)
			}
		}

		o.GroupType = &groupType
		o.GroupBytes = &groupBytes
	}

	return nil
}

func (o Environment) ToCoRIM() (*comid.Environment, error) {
	var err error
	ret := comid.Environment{}
	class := comid.Class{
		Vendor: o.Vendor,
		Model:  o.Model,
		Layer:  o.Layer,
		Index:  o.Index,
	}

	if o.ClassType != nil {
		if o.ClassBytes == nil {
			return nil, errors.New("missing class ID data")
		}

		switch *o.ClassType {
		case comid.OIDType:
			class.ClassID, err = comid.NewOIDClassID(*o.ClassBytes)
			if err != nil {
				return nil, fmt.Errorf("could not initialize OID class ID: %w", err)
			}
		case comid.UUIDType:
			class.ClassID, err = comid.NewUUIDClassID(*o.ClassBytes)
			if err != nil {
				return nil, fmt.Errorf("could not initialize UUID class ID: %w", err)
			}
		case comid.BytesType:
			class.ClassID, err = comid.NewBytesClassID(*o.ClassBytes)
			if err != nil {
				return nil, fmt.Errorf("could not initialize bytes class ID: %w", err)
			}
		default:
			class.ClassID, err = comid.NewClassID(nil, *o.ClassType)
			if err != nil {
				return nil, err
			}

			if err = cbor.Unmarshal(*o.ClassBytes, &class.ClassID.Value); err != nil {
				return nil, fmt.Errorf("could not CBOR-decode class ID: %w", err)
			}
		}
	}

	if class.ClassID != nil || class.Vendor != nil || class.Model != nil ||
		class.Layer != nil || class.Index != nil {
		ret.Class = &class
	}

	if o.InstanceType != nil {
		if o.InstanceBytes == nil {
			return nil, errors.New("missing instance data")
		}

		switch *o.InstanceType {
		case comid.UUIDType:
			ret.Instance, err = comid.NewUUIDInstance(*o.InstanceBytes)
			if err != nil {
				return nil, fmt.Errorf("could not initialize UUID instance: %w", err)
			}
		case "ueid":
			ret.Instance, err = comid.NewUEIDInstance(*o.InstanceBytes)
			if err != nil {
				return nil, fmt.Errorf("could not initialize UEID instance: %w", err)
			}
		case comid.BytesType:
			ret.Instance, err = comid.NewBytesInstance(*o.InstanceBytes)
			if err != nil {
				return nil, fmt.Errorf("could not initialize bytes instance: %w", err)
			}
		default:
			ret.Instance, err = comid.NewInstance(nil, *o.InstanceType)
			if err != nil {
				return nil, err
			}

			if err = cbor.Unmarshal(*o.InstanceBytes, &ret.Instance.Value); err != nil {
				return nil, fmt.Errorf("could not CBOR-decode instance: %w", err)
			}
		}
	}

	if o.GroupType != nil {
		if o.GroupBytes == nil {
			return nil, errors.New("missing group data")
		}

		switch *o.GroupType {
		case comid.UUIDType:
			ret.Group, err = comid.NewUUIDGroup(*o.GroupBytes)
			if err != nil {
				return nil, fmt.Errorf("could not initialize UUID group: %w", err)
			}
		case comid.BytesType:
			ret.Group, err = comid.NewBytesGroup(*o.GroupBytes)
			if err != nil {
				return nil, fmt.Errorf("could not initialize bytes group: %w", err)
			}
		default:
			ret.Group, err = comid.NewGroup(nil, *o.GroupType)
			if err != nil {
				return nil, err
			}

			if err = cbor.Unmarshal(*o.GroupBytes, &ret.Group.Value); err != nil {
				return nil, fmt.Errorf("could not CBOR-decode group: %w", err)
			}
		}
	}

	return &ret, nil
}

func (o *Environment) Validate() error {
	if (o.ClassType == nil) != (o.ClassBytes == nil) {
		return errors.New("ClassType and ClassBytes must be set together")
	}

	if (o.InstanceType == nil) != (o.InstanceBytes == nil) {
		return errors.New("InstanceType and InstanceBytes must be set together")
	}

	if (o.InstanceType == nil) != (o.InstanceBytes == nil) {
		return errors.New("InstanceType and InstanceBytes must be set together")
	}

	if (o.GroupType == nil) != (o.GroupBytes == nil) {
		return errors.New("GroupType and GroupBytes must be set together")
	}

	return nil
}

func (o *Environment) Insert(ctx context.Context, db bun.IDB) error {
	if err := o.Validate(); err != nil {
		return err
	}

	query := db.NewSelect().Model(o)
	err := UpdateSelectQueryFromEnvironment(query, o, true).Scan(ctx)
	if err == sql.ErrNoRows {
		_, err = db.NewInsert().Model(o).Exec(ctx)
	}

	return err
}

func (o *Environment) Select(ctx context.Context, db bun.IDB) error {
	if o.ID == 0 {
		return errors.New("ID not set")
	}

	_, err := db.NewSelect().Model(o).Where("env.id = ?", o.ID).Exec(ctx)
	return err
}

func (o *Environment) DeleteIfOrphaned(ctx context.Context, db bun.IDB) error {
	if o.ID == 0 {
		return errors.New("ID not set")
	}

	owners := []string{"key_triples", "value_triples"}
	for _, table := range owners {
		ownerIDs, err := o.getOwnerIDs(ctx, db, table)
		if err != nil {
			return fmt.Errorf("error getting enviroment owners from %q: %w", table, err)
		}

		if len(ownerIDs) != 0 {
			// not orphaned, so don't delete
			return nil
		}
	}

	_, err := db.NewDelete().Model(o).WherePK().Exec(ctx)
	return err
}

func (o Environment) RenderParts() ([][2]string, error) {
	if o.IsEmpty() {
		return nil, nil
	}

	if err := o.Validate(); err != nil {
		return nil, err
	}

	var ret [][2]string

	if o.Vendor != nil && *o.Vendor != "" {
		ret = append(ret, [2]string{"vendor", *o.Vendor})
	}

	if o.Model != nil && *o.Model != "" {
		ret = append(ret, [2]string{"model", *o.Model})
	}

	if o.ClassBytes != nil {
		var val string
		switch *o.ClassType {
		case comid.OIDType:
			val = comid.OID(*o.ClassBytes).String()
		case comid.UUIDType:
			u, err := uuid.ParseBytes(*o.ClassBytes)
			if err != nil {
				return nil, fmt.Errorf("class: %w", err)
			}
			val = u.String()
		default:
			val = hex.EncodeToString(*o.ClassBytes)
		}

		ret = append(ret, [2]string{"class", val})
	}

	if o.InstanceBytes != nil {
		var val string
		switch *o.InstanceType {
		case comid.OIDType:
			val = comid.OID(*o.InstanceBytes).String()
		case comid.UUIDType:
			u, err := uuid.ParseBytes(*o.InstanceBytes)
			if err != nil {
				return nil, fmt.Errorf("class: %w", err)
			}
			val = u.String()
		case comid.UEIDType:
			val = comid.UEID(*o.InstanceBytes).String()
		default:
			val = hex.EncodeToString(*o.InstanceBytes)
		}

		ret = append(ret, [2]string{"instance", val})
	}

	if o.GroupBytes != nil {
		var val string
		switch *o.GroupType {
		case comid.OIDType:
			val = comid.OID(*o.GroupBytes).String()
		default:
			val = hex.EncodeToString(*o.GroupBytes)
		}

		ret = append(ret, [2]string{"group", val})
	}

	if o.Index != nil {
		ret = append(ret, [2]string{"index", fmt.Sprintf("%d", *o.Index)})
	}

	return ret, nil
}

func (o *Environment) getOwnerIDs(ctx context.Context, db bun.IDB, ownerTable string) ([]int64, error) {
	var ownerIDs []int64
	err := db.NewSelect().
		Model(&ownerIDs).
		Table(ownerTable).
		Column("id").
		Where("environment_id = ?", o.ID).
		Scan(ctx)

	if err == nil || errors.Is(err, sql.ErrNoRows) {
		return ownerIDs, nil
	}

	return nil, err
}

func (o *Environment) IsEmpty() bool {
	return o.ClassType == nil && o.ClassBytes == nil && o.Vendor == nil && o.Model == nil &&
		o.Layer == nil && o.Index == nil && o.InstanceType == nil && o.InstanceBytes == nil &&
		o.GroupType == nil && o.GroupBytes == nil
}

func UpdateSelectQueryFromEnvironment(
	query *bun.SelectQuery,
	env *Environment,
	exact bool,
) *bun.SelectQuery {
	if env.ClassType != nil {
		query.Where("class_type = ?", env.ClassType)
	} else if exact {
		query.Where("class_type IS NULL")
	}

	if env.ClassBytes != nil {
		query.Where("class_bytes = ?", env.ClassBytes)
	} else if exact {
		query.Where("class_bytes IS NULL")
	}

	if env.InstanceType != nil {
		query.Where("instance_type = ?", env.InstanceType)
	} else if exact {
		query.Where("instance_type IS NULL")
	}

	if env.InstanceBytes != nil {
		query.Where("instance_bytes = ?", env.InstanceBytes)
	} else if exact {
		query.Where("instance_bytes IS NULL")
	}

	if env.GroupType != nil {
		query.Where("group_type = ?", env.GroupType)
	} else if exact {
		query.Where("group_type IS NULL")
	}

	if env.GroupBytes != nil {
		query.Where("group_bytes = ?", env.GroupBytes)
	} else if exact {
		query.Where("group_bytes IS NULL")
	}

	if env.Vendor != nil {
		query.Where("vendor = ?", env.Vendor)
	} else if exact {
		query.Where("vendor IS NULL")
	}

	if env.Model != nil {
		query.Where("model = ?", env.Model)
	} else if exact {
		query.Where("model IS NULL")
	}

	if env.Layer != nil {
		query.Where("layer = ?", env.Layer)
	} else if exact {
		query.Where("layer IS NULL")
	}

	if env.Index != nil {
		query.Where("\"index\" = ?", env.Index)
	} else if exact {
		query.Where("\"index\" IS NULL")
	}

	return query
}
