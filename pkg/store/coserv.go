package store

import (
	"errors"
	"fmt"
	"time"

	"github.com/veraison/corim-store/pkg/model"
	"github.com/veraison/corim/comid"
	"github.com/veraison/corim/coserv"
	"github.com/veraison/eat"
)

var ErrMeasuments = errors.New("cannot specify measurements for trust anchors")

// CoSERVService implements CoSERV semantics on top of a Store.
// see https://datatracker.ietf.org/doc/draft-ietf-rats-coserv/
type CoSERVService struct {
	// Store used by this coserive service
	Store *Store
	// FallbackAuthority authority will be used when no other authority can
	// be established for result data.
	FallbackAuthority *comid.CryptoKey
}

// NewCoSERVService creates a new instance of the service.
func NewCoSERVService(store *Store, authority *comid.CryptoKey) *CoSERVService {
	return &CoSERVService{store, authority}
}

// UpdateCoSERV runs the query inside the provided coserv.Coserv object and
// updates its result set with the results.
func (o *CoSERVService) UpdateCoSERV(value *coserv.Coserv) error {
	resultSet, err := o.RunQuery(&value.Profile, &value.Query)
	if err != nil {
		return err
	}

	return value.AddResults(*resultSet)
}

// RunQuery runs the provided coserv.Query, returning the corresponding
// coserv.ResultSet. If profile is specified, only manifests whose profiles
// match will be considered when running the query.
func (o *CoSERVService) RunQuery(profile *eat.Profile, query *coserv.Query) (*coserv.ResultSet, error) {
	result := coserv.NewResultSet()

	var expiry *time.Time

	switch query.ArtifactType {
	case coserv.ArtifactTypeReferenceValues: // nolint:dupl
		queryGroup, err := ValueTripleQueryGroupFromCoSERV(query)
		if err != nil {
			return nil, err
		}

		if profile != nil {
			queryGroup.ForEach(func(v *ValueTripleQuery) {
				v.ProfileFromEAT(profile)
			})
		}

		tripleEntries, err := o.Store.QueryValueTripleEntries(queryGroup)
		if err != nil {
			return nil, err
		}

		triples := make([]*comid.ValueTriple, len(tripleEntries))
		for i, entry := range tripleEntries {
			updateExpiry(&expiry, entry.NotAfter)

			model, err := entry.ToTriple(o.Store.Ctx, o.Store.DB)
			if err != nil {
				return nil, fmt.Errorf("value triple with ID %d: %w", entry.TripleDbID, err)
			}

			triple, err := model.ToCoRIM()
			if err != nil {
				return nil, fmt.Errorf("value triple with ID %d: %w", entry.TripleDbID, err)
			}

			triples[i] = triple
		}

		for _, triple := range triples {
			result.AddReferenceValues(coserv.RefValQuad{
				Authorities: comid.NewCryptoKeys().Add(o.FallbackAuthority),
				RVTriple:    triple,
			})
		}
	case coserv.ArtifactTypeTrustAnchors: // nolint:dupl
		queryGroup, err := KeyTripleQueryGroupFromCoSERV(query)
		if err != nil {
			return nil, err
		}

		if profile != nil {
			queryGroup.ForEach(func(v *KeyTripleQuery) {
				v.ProfileFromEAT(profile)
			})
		}

		tripleEntries, err := o.Store.QueryKeyTripleEntries(queryGroup)
		if err != nil {
			return nil, err
		}

		triples := make([]*comid.KeyTriple, len(tripleEntries))
		for i, entry := range tripleEntries {
			updateExpiry(&expiry, entry.NotAfter)

			model, err := entry.ToTriple(o.Store.Ctx, o.Store.DB)
			if err != nil {
				return nil, fmt.Errorf("key triple with ID %d: %w", entry.TripleDbID, err)
			}

			triple, err := model.ToCoRIM()
			if err != nil {
				return nil, fmt.Errorf("key triple with ID %d: %w", entry.TripleDbID, err)
			}

			triples[i] = triple
		}

		for _, triple := range triples {
			result.AddAttestationKeys(coserv.AKQuad{
				Authorities: comid.NewCryptoKeys().Add(o.FallbackAuthority),
				AKTriple:    triple,
			})
		}
	default:
		return nil, fmt.Errorf("unsupported artifact type: %s", query.ArtifactType.String())
	}

	if expiry != nil {
		result.SetExpiry(*expiry)
	}

	return result, nil
}

func ValueTripleQueryGroupFromCoSERV(cq *coserv.Query) (*ValueTripleQueryGroup, error) { // nolint:dupl
	if cq.ResultType != coserv.ResultTypeCollectedArtifacts {
		return nil, errors.New("only collected results are supported right now")
	}

	var tripleType model.ValueTripleType

	switch cq.ArtifactType {
	case coserv.ArtifactTypeReferenceValues:
		tripleType = model.ReferenceValueTriple
	// TODO(setrofim): while the store supportes endored values, coserv
	// implemementation in corim library doesn't at the time of writing.
	// case coserv.ArtifactTypeEndorsedValues:
	// tripleType = model.EndorsedValueTriple
	default:
		return nil, fmt.Errorf("unsupported triple type: %s", cq.ArtifactType.String())
	}

	ret := NewValueTripleQueryGroup()

	if cq.EnvironmentSelector.Classes != nil {
		for i, statefulClass := range *cq.EnvironmentSelector.Classes {
			query, err := ValueTripleQueryFromStatefulClass(&statefulClass)
			if err != nil {
				return nil, fmt.Errorf("stateful class %d: %w", i, err)
			}

			query.TripleType(tripleType).
				ValidOn(time.Now())

			ret.Add(query)
		}
	}

	if cq.EnvironmentSelector.Instances != nil {
		for i, statefulInstance := range *cq.EnvironmentSelector.Instances {
			query, err := ValueTripleQueryFromStatefulInstance(&statefulInstance)
			if err != nil {
				return nil, fmt.Errorf("stateful instance %d: %w", i, err)
			}

			query.TripleType(tripleType).
				ValidOn(time.Now())

			ret.Add(query)
		}
	}

	if cq.EnvironmentSelector.Groups != nil {
		for i, statefulGroup := range *cq.EnvironmentSelector.Groups {
			query, err := ValueTripleQueryFromStatefulGroup(&statefulGroup)
			if err != nil {
				return nil, fmt.Errorf("stateful instance %d: %w", i, err)
			}

			query.TripleType(tripleType).
				ValidOn(time.Now())

			ret.Add(query)
		}
	}

	return ret, nil
}

func ValueTripleQueryFromStatefulClass(statefulClass *coserv.StatefulClass) (*ValueTripleQuery, error) {
	query := NewValueTripleQuery().Environment(func(e *EnvironmentQuery) {
		e.Class(func(cs *ClassSubquery) {
			cs.UpdateFromCoRIM(statefulClass.Class)
		})
	})

	if statefulClass.Measurements != nil {
		for i, measurement := range statefulClass.Measurements.Values {
			measurementModel, err := model.NewMeasurementFromCoRIM(&measurement)
			if err != nil {
				return nil, fmt.Errorf("measurement %d: %w", i, err)
			}

			query.Measurement(func(e *MeasurementQuery) {
				e.UpdateFromModel(measurementModel)
			})
		}
	}

	return query, nil
}

func ValueTripleQueryFromStatefulInstance(statefulInstance *coserv.StatefulInstance) (*ValueTripleQuery, error) {
	query := NewValueTripleQuery().Environment(func(e *EnvironmentQuery) {
		e.Instance(statefulInstance.Instance.Type(), statefulInstance.Instance.Bytes())
	})

	if statefulInstance.Measurements != nil {
		for i, measurement := range statefulInstance.Measurements.Values {
			measurementModel, err := model.NewMeasurementFromCoRIM(&measurement)
			if err != nil {
				return nil, fmt.Errorf("measurement %d: %w", i, err)
			}

			query.Measurement(func(e *MeasurementQuery) {
				e.UpdateFromModel(measurementModel)
			})
		}
	}

	return query, nil
}

func ValueTripleQueryFromStatefulGroup(statefulGroup *coserv.StatefulGroup) (*ValueTripleQuery, error) {
	query := NewValueTripleQuery().Environment(func(e *EnvironmentQuery) {
		e.Group(statefulGroup.Group.Type(), statefulGroup.Group.Bytes())
	})

	if statefulGroup.Measurements != nil {
		for i, measurement := range statefulGroup.Measurements.Values {
			measurementModel, err := model.NewMeasurementFromCoRIM(&measurement)
			if err != nil {
				return nil, fmt.Errorf("measurement %d: %w", i, err)
			}

			query.Measurement(func(e *MeasurementQuery) {
				e.UpdateFromModel(measurementModel)
			})
		}
	}

	return query, nil
}

func KeyTripleQueryGroupFromCoSERV(cq *coserv.Query) (*KeyTripleQueryGroup, error) { // nolint:dupl
	if cq.ResultType != coserv.ResultTypeCollectedArtifacts {
		return nil, errors.New("only collected results are supported right now")
	}

	var tripleType model.KeyTripleType

	switch cq.ArtifactType {
	case coserv.ArtifactTypeTrustAnchors:
		tripleType = model.AttestKeyTriple
	default:
		return nil, fmt.Errorf("unsupported triple type: %s", cq.ArtifactType.String())
	}

	ret := NewKeyTripleQueryGroup()

	if cq.EnvironmentSelector.Classes != nil {
		for i, statefulClass := range *cq.EnvironmentSelector.Classes {
			query, err := KeyTripleQueryFromStatefulClass(&statefulClass)
			if err != nil {
				return nil, fmt.Errorf("stateful class %d: %w", i, err)
			}

			query.TripleType(tripleType).
				ValidOn(time.Now())

			ret.Add(query)
		}
	}

	if cq.EnvironmentSelector.Instances != nil {
		for i, statefulInstance := range *cq.EnvironmentSelector.Instances {
			query, err := KeyTripleQueryFromStatefulInstance(&statefulInstance)
			if err != nil {
				return nil, fmt.Errorf("stateful instance %d: %w", i, err)
			}

			query.TripleType(tripleType).
				ValidOn(time.Now())
			ret.Add(query)
		}
	}

	if cq.EnvironmentSelector.Groups != nil {
		for i, statefulGroup := range *cq.EnvironmentSelector.Groups {
			query, err := KeyTripleQueryFromStatefulGroup(&statefulGroup)
			if err != nil {
				return nil, fmt.Errorf("stateful instance %d: %w", i, err)
			}

			query.TripleType(tripleType).
				ValidOn(time.Now())

			ret.Add(query)
		}
	}

	return ret, nil
}

func KeyTripleQueryFromStatefulClass(statefulClass *coserv.StatefulClass) (*KeyTripleQuery, error) {
	query := NewKeyTripleQuery().Environment(func(e *EnvironmentQuery) {
		e.Class(func(cs *ClassSubquery) {
			cs.UpdateFromCoRIM(statefulClass.Class)
		})
	})

	if statefulClass.Measurements != nil {
		return nil, ErrMeasuments
	}

	return query, nil
}

func KeyTripleQueryFromStatefulInstance(statefulInstance *coserv.StatefulInstance) (*KeyTripleQuery, error) {
	query := NewKeyTripleQuery().Environment(func(e *EnvironmentQuery) {
		e.Instance(statefulInstance.Instance.Type(), statefulInstance.Instance.Bytes())
	})

	if statefulInstance.Measurements != nil {
		return nil, ErrMeasuments
	}

	return query, nil
}

func KeyTripleQueryFromStatefulGroup(statefulGroup *coserv.StatefulGroup) (*KeyTripleQuery, error) {
	query := NewKeyTripleQuery().Environment(func(e *EnvironmentQuery) {
		e.Group(statefulGroup.Group.Type(), statefulGroup.Group.Bytes())
	})

	if statefulGroup.Measurements != nil {
		return nil, ErrMeasuments
	}

	return query, nil
}

func updateExpiry(expiry **time.Time, value *time.Time) {
	if *expiry == nil {
		*expiry = value
		return
	}

	if value == nil {
		return
	}

	if value.Before(**expiry) {
		*expiry = value
	}
}
