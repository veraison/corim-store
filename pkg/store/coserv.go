package store

import (
	"errors"
	"fmt"
	"time"

	"github.com/veraison/cmw"
	"github.com/veraison/corim-store/pkg/model"
	"github.com/veraison/corim-store/pkg/util"
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
	// MaxExpiry is the maximum limit on the expiry of a result set. When a
	// result set is created, its expiry is set based on the validity of
	// the CoRIMs involved in creating it. If that expiry is farther into the
	// future than MaxExpiry (or if the CoRIMs did not specify validity),
	// expiry will be set to MaxExpiry from the time of result set creation
	// instead.
	MaxExpiry time.Duration
}

// NewCoSERVService creates a new instance of the service.
func NewCoSERVService(store *Store, authority *comid.CryptoKey, maxExpiry time.Duration) *CoSERVService {
	return &CoSERVService{store, authority, maxExpiry}
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
	if err := query.Valid(); err != nil {
		return nil, fmt.Errorf("invalid query: %w", err)
	}

	var result *coserv.ResultSet
	var err error
	var expiry *time.Time

	if query.EnvironmentSelector != nil {
		result, expiry, err = o.runEnvironmentQuery(profile, query)
	} else {
		result, expiry, err = o.runRIMQuery(profile, query)
	}

	if err != nil {
		if errors.Is(err, ErrNoMatch) {
			result = coserv.NewResultSet()
		} else {
			return nil, err
		}
	}

	if expiry != nil {
		result.SetExpiry(earliest(*expiry, time.Now().Add(o.MaxExpiry)))
	} else {
		result.SetExpiry(time.Now().Add(o.MaxExpiry))
	}

	return result, nil
}

func (o *CoSERVService) runEnvironmentQuery(
	profile *eat.Profile,
	query *coserv.Query,
) (*coserv.ResultSet, *time.Time, error) {
	var expiry *time.Time
	result := coserv.NewResultSet()

	switch *query.ArtifactType {
	case coserv.ArtifactTypeReferenceValues: // nolint:dupl
		queryGroup, err := ValueTripleQueryGroupFromCoSERV(query)
		if err != nil {
			return nil, nil, err
		}

		if profile != nil {
			queryGroup.ForEach(func(v *ValueTripleQuery) {
				v.ProfileFromEAT(profile)
			})
		}

		tripleEntries, err := o.Store.QueryValueTripleEntries(queryGroup)
		if err != nil {
			return nil, nil, err
		}

		triples := make([]*comid.ValueTriple, len(tripleEntries))
		for i, entry := range tripleEntries {
			updateExpiry(&expiry, entry.NotAfter)

			model, err := entry.ToTriple(o.Store.Ctx, o.Store.DB)
			if err != nil {
				return nil, nil, fmt.Errorf("value triple with ID %d: %w", entry.TripleDbID, err)
			}

			triple, err := model.ToCoRIM()
			if err != nil {
				return nil, nil, fmt.Errorf("value triple with ID %d: %w", entry.TripleDbID, err)
			}

			triples[i] = triple
		}

		for _, triple := range triples {
			result.AddReferenceValues(coserv.RefValQuad{
				Authorities: comid.NewCryptoKeys().Add(o.FallbackAuthority),
				RVTriple:    triple,
			})
		}
	case coserv.ArtifactTypeEndorsedValues: // nolint:dupl
		queryGroup, err := ValueTripleQueryGroupFromCoSERV(query)
		if err != nil {
			return nil, nil, err
		}

		condQueryGroup, err := ConditionalEndorsementTripleQueryGroupFromCoSERV(query)
		if err != nil && !errors.Is(err, ErrNoMatch) {
			return nil, nil, err
		}

		if profile != nil {
			queryGroup.ForEach(func(v *ValueTripleQuery) {
				v.ProfileFromEAT(profile)
			})

			condQueryGroup.ForEach(func(v *ConditionalEndorsementTripleQuery) {
				v.ProfileFromEAT(profile)
			})
		}

		tripleEntries, err := o.Store.QueryValueTripleEntries(queryGroup)
		if err != nil && !errors.Is(err, ErrNoMatch) {
			return nil, nil, err
		}

		condTripleEntries, err := o.Store.QueryConditionalEndorsementTripleEntries(condQueryGroup)
		if err != nil && !errors.Is(err, ErrNoMatch) {
			return nil, nil, err
		}

		if len(tripleEntries) == 0 && len(condTripleEntries) == 0 {
			return nil, nil, ErrNoMatch
		}

		triples := make([]*comid.ValueTriple, len(tripleEntries))
		for i, entry := range tripleEntries {
			updateExpiry(&expiry, entry.NotAfter)

			model, err := entry.ToTriple(o.Store.Ctx, o.Store.DB)
			if err != nil {
				return nil, nil, fmt.Errorf("value triple with ID %d: %w", entry.TripleDbID, err)
			}

			triple, err := model.ToCoRIM()
			if err != nil {
				return nil, nil, fmt.Errorf("value triple with ID %d: %w", entry.TripleDbID, err)
			}

			triples[i] = triple
		}

		for _, triple := range triples {
			result.AddEndorsedValues(coserv.EndValQuad{
				Authorities: comid.NewCryptoKeys().Add(o.FallbackAuthority),
				EVTriple:    triple,
			})
		}

		condTriples := make([]*comid.CondEndorseTriple, len(condTripleEntries))
		for i, entry := range condTripleEntries {
			updateExpiry(&expiry, entry.NotAfter)

			model, err := entry.ToTriple(o.Store.Ctx, o.Store.DB)
			if err != nil {
				return nil, nil, fmt.Errorf("conditional endorsement triple with ID %d: %w",
					entry.TripleDbID, err)
			}

			triple, err := model.ToCoRIM()
			if err != nil {
				return nil, nil, fmt.Errorf("conditional endorsement triple with ID %d: %w",
					entry.TripleDbID, err)
			}

			condTriples[i] = triple
		}

		for _, triple := range condTriples {
			result.AddConditionalEndorsementValues(coserv.CondEndValQuad{
				Authorities: comid.NewCryptoKeys().Add(o.FallbackAuthority),
				CETriple:    triple,
			})
		}
	case coserv.ArtifactTypeTrustAnchors: // nolint:dupl
		queryGroup, err := KeyTripleQueryGroupFromCoSERV(query)
		if err != nil {
			return nil, nil, err
		}

		if profile != nil {
			queryGroup.ForEach(func(v *KeyTripleQuery) {
				v.ProfileFromEAT(profile)
			})
		}

		tripleEntries, err := o.Store.QueryKeyTripleEntries(queryGroup)
		if err != nil {
			return nil, nil, err
		}

		triples := make([]*comid.KeyTriple, len(tripleEntries))
		for i, entry := range tripleEntries {
			updateExpiry(&expiry, entry.NotAfter)

			model, err := entry.ToTriple(o.Store.Ctx, o.Store.DB)
			if err != nil {
				return nil, nil, fmt.Errorf("key triple with ID %d: %w", entry.TripleDbID, err)
			}

			triple, err := model.ToCoRIM()
			if err != nil {
				return nil, nil, fmt.Errorf("key triple with ID %d: %w", entry.TripleDbID, err)
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
		return nil, nil, fmt.Errorf("unsupported artifact type: %s", query.ArtifactType.String())
	}

	return result, expiry, nil
}

func (o *CoSERVService) runRIMQuery(
	profile *eat.Profile,
	query *coserv.Query,
) (*coserv.ResultSet, *time.Time, error) {
	if query.RimSelector == nil {
		return nil, nil, errors.New("no RIM selectors specified")
	}

	rimCollection, err := cmw.NewCollection("")
	if err != nil {
		return nil, nil, fmt.Errorf("CMW collection: %w", err)
	}

	var expiry *time.Time

	var manifestID string
	var manifestDbID int64
	var notAfter *time.Time
	for i, selector := range *query.RimSelector {
		switch selector.Type {
		case coserv.RimSelectorTypeCorim:
			query := NewManifestQuery().
				ManifestIDFromSWID(selector.TagID).
				ProfileFromEAT(profile).
				ValidOn(time.Now())

			entries, err := o.Store.QueryManifestEntries(query)
			if err != nil {
				if errors.Is(err, ErrNoMatch) {
					continue
				}

				return nil, nil, fmt.Errorf("RIM selector %d: %w", i, err)
			}

			if len(entries) > 1 {
				return nil, nil, fmt.Errorf("non-unique Manifest ID %v", selector.TagID)
			}

			manifestID, manifestDbID = entries[0].ManifestID, entries[0].ManifestDbID
			notAfter = entries[0].NotAfter
		case coserv.RimSelectorTypeComid:
			query := NewModuleTagQuery().
				ModuleTagIDFromSWID(selector.TagID).
				ProfileFromEAT(profile).
				ValidOn(time.Now())

			entries, err := o.Store.QueryModuleTagEntries(query)
			if err != nil {
				if errors.Is(err, ErrNoMatch) {
					continue
				}

				return nil, nil, fmt.Errorf("RIM selector %d: %w", i, err)
			}

			selectedModuleTag := entries[0]
			if len(entries) > 1 {
				for _, moduleTag := range entries[1:] {
					if moduleTag.ModuleTagVersion > selectedModuleTag.ModuleTagVersion {
						selectedModuleTag = moduleTag
					}
				}
			}

			manifestID, manifestDbID = selectedModuleTag.ManifestID, selectedModuleTag.ManifestDbID
			notAfter = entries[0].NotAfter
		case coserv.RimSelectorTypeCoswid:
			return nil, nil, errors.New("CoSWID selectors not supported")
		default:
			return nil, nil, fmt.Errorf("unknown selector")
		}

		updateExpiry(&expiry, notAfter)

		cmwEntry, err := o.getRIMEntryFromID(profile, manifestID, manifestDbID)
		if err != nil {
			return nil, nil, fmt.Errorf("manifest %q: %w", manifestID, err)
		}

		if err := rimCollection.AddCollectionItem(manifestID, cmwEntry); err != nil {
			return nil, nil, fmt.Errorf("CMW entry %q: %w", manifestID, err)
		}
	}

	result := coserv.NewResultSet().SetRIMs(*rimCollection)
	return result, expiry, nil
}

func (o *CoSERVService) getRIMEntryFromID(
	profile *eat.Profile,
	manifestID string,
	dbID int64,
) (*cmw.CMW, error) {
	bytes, err := o.Store.GetTokenBytes(manifestID)
	if err != nil {
		if err != ErrNoMatch {
			return nil, err
		}

		manifest := model.Manifest{ID: dbID}
		if err := manifest.Select(o.Store.Ctx, o.Store.DB); err != nil {
			return nil, err
		}

		unsigned, err := manifest.ToCoRIM()
		if err != nil {
			return nil, err
		}

		bytes, err = unsigned.ToCBOR()
		if err != nil {
			return nil, err
		}
	}

	contentType := "application/rim+cbor"
	if util.IsSignedCoRIM(bytes) {
		contentType = "application/rim+cose"
	}

	if profile != nil && (profile.IsURI() || profile.IsOID()) {
		profileValue, err := profile.Get()
		if err != nil {
			return nil, fmt.Errorf("profile: %w", err)
		}

		contentType = fmt.Sprintf("%s; profile=%q", contentType, profileValue)
	}

	return cmw.NewMonad(contentType, bytes)
}

func ValueTripleQueryGroupFromCoSERV(cq *coserv.Query) (*ValueTripleQueryGroup, error) { // nolint:dupl
	if cq.ResultType == nil {
		return nil, errors.New("result type not set")
	}

	if cq.ArtifactType == nil {
		return nil, errors.New("artifact type not set")
	}

	if *cq.ResultType != coserv.ResultTypeCollectedArtifacts {
		return nil, errors.New("only collected results are supported right now")
	}

	var tripleType model.ValueTripleType

	switch *cq.ArtifactType {
	case coserv.ArtifactTypeReferenceValues:
		tripleType = model.ReferenceValueTriple
	case coserv.ArtifactTypeEndorsedValues:
		tripleType = model.EndorsedValueTriple
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

func ConditionalEndorsementTripleQueryGroupFromCoSERV(cq *coserv.Query) (*ConditionalEndorsementTripleQueryGroup, error) { // nolint:dupl
	if cq.ResultType == nil {
		return nil, errors.New("result type not set")
	}

	if cq.ArtifactType == nil {
		return nil, errors.New("artifact type not set")
	}

	if *cq.ResultType != coserv.ResultTypeCollectedArtifacts {
		return nil, errors.New("only collected results are supported right now")
	}

	ret := NewConditionalEndorsementTripleQueryGroup()

	if cq.EnvironmentSelector.Classes != nil {
		for i, statefulClass := range *cq.EnvironmentSelector.Classes {
			query, err := ConditionalEndorsementTripleQueryFromStatefulClass(&statefulClass)
			if err != nil {
				return nil, fmt.Errorf("stateful class %d: %w", i, err)
			}

			query.ValidOn(time.Now())

			ret.Add(query)
		}
	}

	if cq.EnvironmentSelector.Instances != nil {
		for i, statefulInstance := range *cq.EnvironmentSelector.Instances {
			query, err := ConditionalEndorsementTripleQueryFromStatefulInstance(&statefulInstance)
			if err != nil {
				return nil, fmt.Errorf("stateful instance %d: %w", i, err)
			}

			query.ValidOn(time.Now())

			ret.Add(query)
		}
	}

	if cq.EnvironmentSelector.Groups != nil {
		for i, statefulGroup := range *cq.EnvironmentSelector.Groups {
			query, err := ConditionalEndorsementTripleQueryFromStatefulGroup(&statefulGroup)
			if err != nil {
				return nil, fmt.Errorf("stateful instance %d: %w", i, err)
			}

			query.ValidOn(time.Now())

			ret.Add(query)
		}
	}

	return ret, nil
}

func ConditionalEndorsementTripleQueryFromStatefulClass(
	statefulClass *coserv.StatefulClass,
) (*ConditionalEndorsementTripleQuery, error) {
	var measurementModels []*model.Measurement
	if statefulClass.Measurements != nil {
		measurementModels = make([]*model.Measurement, len(statefulClass.Measurements.Values))

		for i, measurement := range statefulClass.Measurements.Values {
			measurementModel, err := model.NewMeasurementFromCoRIM(&measurement)
			if err != nil {
				return nil, fmt.Errorf("measurement %d: %w", i, err)
			}
			measurementModels[i] = measurementModel
		}
	}

	query := NewConditionalEndorsementTripleQuery().Condition(func(seq *StatefulEnvironmentQuery) {
		seq.Class(func(cs *ClassSubquery) {
			cs.UpdateFromCoRIM(statefulClass.Class)
		})

		for _, measurementModel := range measurementModels {
			seq.Measurement(func(e *MeasurementQuery) {
				e.UpdateFromModel(measurementModel)
			})
		}
	})

	return query, nil
}

func ConditionalEndorsementTripleQueryFromStatefulInstance(
	statefulInstance *coserv.StatefulInstance,
) (*ConditionalEndorsementTripleQuery, error) {
	var measurementModels []*model.Measurement
	if statefulInstance.Measurements != nil {
		measurementModels = make([]*model.Measurement, len(statefulInstance.Measurements.Values))

		for i, measurement := range statefulInstance.Measurements.Values {
			measurementModel, err := model.NewMeasurementFromCoRIM(&measurement)
			if err != nil {
				return nil, fmt.Errorf("measurement %d: %w", i, err)
			}
			measurementModels[i] = measurementModel
		}
	}

	query := NewConditionalEndorsementTripleQuery().Condition(func(seq *StatefulEnvironmentQuery) {
		seq.Instance(statefulInstance.Instance.Type(), statefulInstance.Instance.Bytes())

		for _, measurementModel := range measurementModels {
			seq.Measurement(func(e *MeasurementQuery) {
				e.UpdateFromModel(measurementModel)
			})
		}
	})

	return query, nil
}

func ConditionalEndorsementTripleQueryFromStatefulGroup(
	statefulGroup *coserv.StatefulGroup,
) (*ConditionalEndorsementTripleQuery, error) {
	var measurementModels []*model.Measurement
	if statefulGroup.Measurements != nil {
		measurementModels = make([]*model.Measurement, len(statefulGroup.Measurements.Values))

		for i, measurement := range statefulGroup.Measurements.Values {
			measurementModel, err := model.NewMeasurementFromCoRIM(&measurement)
			if err != nil {
				return nil, fmt.Errorf("measurement %d: %w", i, err)
			}
			measurementModels[i] = measurementModel
		}
	}

	query := NewConditionalEndorsementTripleQuery().Condition(func(seq *StatefulEnvironmentQuery) {
		seq.Group(statefulGroup.Group.Type(), statefulGroup.Group.Bytes())

		for _, measurementModel := range measurementModels {
			seq.Measurement(func(e *MeasurementQuery) {
				e.UpdateFromModel(measurementModel)
			})
		}
	})

	return query, nil
}

func KeyTripleQueryGroupFromCoSERV(cq *coserv.Query) (*KeyTripleQueryGroup, error) { // nolint:dupl
	if cq.ResultType == nil {
		return nil, errors.New("result type not set")
	}

	if cq.ArtifactType == nil {
		return nil, errors.New("artifact type not set")
	}

	if *cq.ResultType != coserv.ResultTypeCollectedArtifacts {
		return nil, errors.New("only collected results are supported right now")
	}

	var tripleType model.KeyTripleType

	switch *cq.ArtifactType {
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

func earliest(lhs, rhs time.Time) time.Time {
	if rhs.Before(lhs) {
		return rhs
	}

	return lhs
}
