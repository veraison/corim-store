package store

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/veraison/corim-store/pkg/model"
	"github.com/veraison/corim/comid"
	"github.com/veraison/corim/coserv"
	"github.com/veraison/eat"
)

func TestCoSERVService(t *testing.T) {
	ctx := context.Background()
	db := model.NewTestDBWithFixtures(t, map[string][]byte{
		"cryptokeys.yaml":         cryptoKeysFixture,
		"digests.yaml":            digestsFixture,
		"manifests.yaml":          manifestsFixture,
		"module_tags.yaml":        moduleTagsFixture,
		"triples.yaml":            triplesFixture,
		"environments.yaml":       environmentsFixture,
		"measurements.yaml":       measurementsFixture,
		"measurement_values.yaml": measurementValuesFixture,
	})
	defer func() { assert.NoError(t, db.Close()) }()

	store, err := OpenWithDB(ctx, db)
	require.NoError(t, err)

	authority, err := comid.NewCryptoKeyTaggedBytes("test")
	require.NoError(t, err)

	service := NewCoSERVService(store, authority)

	profile, err := eat.NewProfile("http://example.com")
	require.NoError(t, err)

	name := "foo"

	cs := &coserv.Coserv{
		Profile: *profile,
		Query: coserv.Query{
			ArtifactType: coserv.ArtifactTypeReferenceValues,
			EnvironmentSelector: *coserv.NewEnvironmentSelector().
				AddClass(coserv.StatefulClass{
					Class: comid.NewClassBytes([]byte{
						0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07,
						0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07,
						0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07,
						0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07,
					}),
					Measurements: comid.NewMeasurements().Add(&comid.Measurement{
						Val: comid.Mval{Name: &name},
					}),
				}),
			Timestamp:  time.Now(),
			ResultType: coserv.ResultTypeCollectedArtifacts,
		},
	}

	err = service.UpdateCoSERV(cs)
	assert.NoError(t, err)
	assert.Equal(t, "0.1.2.3.4", (*cs.Results.RVQ)[0].RVTriple.Measurements.Values[0].Key.Value.String())

	cs = &coserv.Coserv{
		Profile: *profile,
		Query: coserv.Query{
			ArtifactType: coserv.ArtifactTypeReferenceValues,
			EnvironmentSelector: *coserv.NewEnvironmentSelector().
				AddInstance(coserv.StatefulInstance{
					Instance: comid.MustNewBytesInstance([]byte{
						0x10, 0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17,
						0x10, 0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17,
						0x10, 0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17,
						0x10, 0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17,
					}),
					Measurements: comid.NewMeasurements().Add(&comid.Measurement{
						Val: comid.Mval{Name: &name},
					}),
				}),
			ResultType: coserv.ResultTypeCollectedArtifacts,
		},
	}

	err = service.UpdateCoSERV(cs)
	assert.NoError(t, err)
	assert.Equal(t, "0.1.2.3.4", (*cs.Results.RVQ)[0].RVTriple.Measurements.Values[0].Key.Value.String())
	assert.Equal(t, 2999, cs.Results.Expiry.Year())

	cs = &coserv.Coserv{
		Profile: *profile,
		Query: coserv.Query{
			ArtifactType: coserv.ArtifactTypeReferenceValues,
			EnvironmentSelector: *coserv.NewEnvironmentSelector().
				AddGroup(coserv.StatefulGroup{
					Group: comid.MustNewBytesGroup([]byte{
						0x20, 0x21, 0x22, 0x23, 0x24, 0x25, 0x26, 0x27,
						0x20, 0x21, 0x22, 0x23, 0x24, 0x25, 0x26, 0x27,
						0x20, 0x21, 0x22, 0x23, 0x24, 0x25, 0x26, 0x27,
						0x20, 0x21, 0x22, 0x23, 0x24, 0x25, 0x26, 0x27,
					}),
					Measurements: comid.NewMeasurements().Add(&comid.Measurement{
						Val: comid.Mval{Name: &name},
					}),
				}),
			ResultType: coserv.ResultTypeCollectedArtifacts,
		},
	}

	err = service.UpdateCoSERV(cs)
	assert.NoError(t, err)
	assert.Equal(t, "0.1.2.3.4", (*cs.Results.RVQ)[0].RVTriple.Measurements.Values[0].Key.Value.String())

	cs = &coserv.Coserv{
		Profile: *profile,
		Query: coserv.Query{
			ArtifactType: coserv.ArtifactTypeReferenceValues,
			EnvironmentSelector: *coserv.NewEnvironmentSelector().
				AddGroup(coserv.StatefulGroup{
					Group: comid.MustNewBytesGroup([]byte{
						0x20, 0x21, 0x22, 0x23, 0x24, 0x25, 0x26, 0x27,
						0x20, 0x21, 0x22, 0x23, 0x24, 0x25, 0x26, 0x27,
						0x20, 0x21, 0x22, 0x23, 0x24, 0x25, 0x26, 0x27,
						0x20, 0x21, 0x22, 0x23, 0x24, 0x25, 0x26, 0x27,
					}),
				}),
			ResultType: coserv.ResultTypeCollectedArtifacts,
		},
	}

	err = service.UpdateCoSERV(cs)
	assert.NoError(t, err)
	assert.Equal(t, "0.1.2.3.4", (*cs.Results.RVQ)[0].RVTriple.Measurements.Values[0].Key.Value.String())

	cs = &coserv.Coserv{
		Profile: *profile,
		Query: coserv.Query{
			ArtifactType: coserv.ArtifactTypeTrustAnchors,
			EnvironmentSelector: *coserv.NewEnvironmentSelector().
				AddClass(coserv.StatefulClass{
					Class: comid.NewClassUUID(comid.UUID{
						0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07,
						0x80, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07,
					}),
				}),
			ResultType: coserv.ResultTypeCollectedArtifacts,
		},
	}

	err = service.UpdateCoSERV(cs)
	assert.NoError(t, err)
	expected := comid.TaggedPKIXBase64Key("-----BEGIN PUBLIC KEY-----\nMFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAEW1BvqF+/ry8BWa7ZEMU1xYYHEQ8B\nlLT4MFHOaO+ICTtIvrEeEpr/sfTAP66H2hCHdb5HEXKtRKod6QLcOLPA1Q==\n-----END PUBLIC KEY-----\n")
	assert.Equal(t, expected, (*cs.Results.AKQ)[0].AKTriple.VerifKeys[0].Value)

	cs = &coserv.Coserv{
		Profile: *profile,
		Query: coserv.Query{
			ArtifactType: coserv.ArtifactTypeTrustAnchors,
			EnvironmentSelector: *coserv.NewEnvironmentSelector().
				AddGroup(coserv.StatefulGroup{
					Group: comid.MustNewBytesGroup([]byte{
						0x20, 0x21, 0x22, 0x23, 0x24, 0x25, 0x26, 0x27,
						0x20, 0x21, 0x22, 0x23, 0x24, 0x25, 0x26, 0x27,
						0x20, 0x21, 0x22, 0x23, 0x24, 0x25, 0x26, 0x27,
						0x20, 0x21, 0x22, 0x23, 0x24, 0x25, 0x26, 0x27,
					}),
				}),
			ResultType: coserv.ResultTypeCollectedArtifacts,
		},
	}

	err = service.UpdateCoSERV(cs)
	assert.NoError(t, err)
	assert.Equal(t, expected, (*cs.Results.AKQ)[0].AKTriple.VerifKeys[0].Value)

	cs = &coserv.Coserv{
		Profile: *profile,
		Query: coserv.Query{
			ArtifactType: coserv.ArtifactTypeTrustAnchors,
			EnvironmentSelector: *coserv.NewEnvironmentSelector().
				AddClass(coserv.StatefulClass{
					Class: comid.NewClassUUID(comid.UUID{
						0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07,
						0x80, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07,
					}),
					Measurements: comid.NewMeasurements().Add(&comid.Measurement{
						Val: comid.Mval{Name: &name},
					}),
				}),
			ResultType: coserv.ResultTypeCollectedArtifacts,
		},
	}

	err = service.UpdateCoSERV(cs)
	assert.ErrorIs(t, err, ErrMeasuments)
}
