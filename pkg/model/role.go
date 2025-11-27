package model

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strconv"

	"github.com/uptrace/bun"
	"github.com/veraison/corim/comid"
	"github.com/veraison/corim/corim"
)

func CoMIDRolesFromCoRIM(origin comid.Roles) ([]RoleEntry, error) {
	ret := make([]RoleEntry, 0, len(origin))

	for _, role := range origin {
		ret = append(ret, RoleEntry{Role: role.String()})
	}

	return ret, nil
}

func CoRIMRolesFromCoRIM(origin corim.Roles) ([]RoleEntry, error) {
	ret := make([]RoleEntry, 0, len(origin))

	for _, role := range origin {
		ret = append(ret, RoleEntry{Role: role.String()})
	}

	return ret, nil
}

var roleRegex = regexp.MustCompile(`Role\((\d)\)`)

func ParseCoRIMRole(text string) (corim.Role, error) {
	switch text {
	case "manifestCreator":
		return corim.RoleManifestCreator, nil
	case "manifestSigner":
		return corim.RoleManifestSigner, nil
	default:
		matches := roleRegex.FindStringSubmatch(text)
		switch len(matches) {
		case 2:
			role, err := strconv.Atoi(matches[1])
			return corim.Role(role), err
		default:
			return corim.Role(0), fmt.Errorf("invalid CoRIM role: %s", text)
		}
	}
}

func ParseCoMIDRole(text string) (comid.Role, error) {
	switch text {
	case "tagCreator":
		return comid.RoleTagCreator, nil
	case "creator":
		return comid.RoleCreator, nil
	case "maintainer":
		return comid.RoleMaintainer, nil
	default:
		matches := roleRegex.FindStringSubmatch(text)
		switch len(matches) {
		case 2:
			role, err := strconv.Atoi(matches[1])
			return comid.Role(role), err
		default:
			return comid.Role(0), fmt.Errorf("invalid CoMID role: %s", text)
		}
	}
}

type RoleEntry struct {
	bun.BaseModel `bun:"table:roles,alias:rol"`

	ID int64 `bun:",pk,autoincrement"`

	Role string

	EntityID int64
}

func MustNewCoRIMRoleEntry(text string) *RoleEntry {
	ret, err := NewCoRIMRoleEntry(text)
	if err != nil {
		panic(err)
	}

	return ret
}

func NewCoRIMRoleEntry(text string) (*RoleEntry, error) {
	if _, err := ParseCoRIMRole(text); err != nil {
		return nil, err
	}

	ret := RoleEntry{Role: text}

	return &ret, nil
}

func MustNewCoMIDRoleEntry(text string) *RoleEntry {
	ret, err := NewCoMIDRoleEntry(text)
	if err != nil {
		panic(err)
	}

	return ret
}

func NewCoMIDRoleEntry(text string) (*RoleEntry, error) {
	if _, err := ParseCoMIDRole(text); err != nil {
		return nil, err
	}

	ret := RoleEntry{Role: text}

	return &ret, nil
}

func (o *RoleEntry) Insert(ctx context.Context, db bun.IDB) error {
	_, err := db.NewInsert().Model(o).Ignore().Exec(ctx)
	return err
}

func (o *RoleEntry) Select(ctx context.Context, db bun.IDB) error {
	if o.ID == 0 {
		return errors.New("ID not set")
	}

	_, err := db.NewSelect().Model(o).Where("rol.id = ?", o.ID).Exec(ctx)
	return err
}

func (o *RoleEntry) Delete(ctx context.Context, db bun.IDB) error {
	if o.ID == 0 {
		return errors.New("ID not set")
	}

	_, err := db.NewDelete().Model(o).WherePK().Exec(ctx)
	return err
}
