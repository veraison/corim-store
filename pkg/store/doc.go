/*
Package store implements a repository for reference values, endorsements and
trust anchors extracted from CoRIMs. Stored values can then be queried and
retrieved. The repository is implemented on top of an SQL Database Management
System (sqlite3, PostgreSQL, and MariaDB/MySQL are supported).

A new Store is created using configuration that contains database connection
settings and options that determine the behavior of the store:

	cfg := store.NewConfig(
	    "sqlite3", "file::memory:", // database connection settings
	    store.OptionRequireLabel, // require non-empty labels for added values
	    store.OptionSHA256, // use SHA256 for internal hashes
	)

	repo, err := store.Open(context.Background(), cfg)
	if err != nil {
	    return err
	}

If the database has not yet been initialized, this can be done via the Store
object:

	if err := repo.Init(); err != nil {
	    return err
	}

This only needs to be done once for a new database if it has not been
initialized by other means (e.g. via CLI).

The store can be populated using CBOR-encoded CoRIMs:

	bytes, err := os.ReadFile("sample/corim/unsigned-cca-ta.cbor")
	if err != nil {
	    return err
	}

	if err := repo.AddBytes(bytes, "mylabel", true); err != nil {
	    return err
	}

The label provides a "namespace" for added values. It can be omitted by
specifying an empty string, unless the store was opened with
`store.OptionRequireLabel`. The last boolean argument indicates whether the
added values should be "activated" making them available to verifiers.

Parsed CoRIMs can also be added:

	uc, err := corim.UnmarshalAndValidateUnsignedCorimFromCBOR(bytes)
	if err != nil {
	    return err
	}

	err := repo.AddCoRIM(uc, sha256.Sum256(bytes), "mylabel", true);
	if err != nil {
	    return err
	}

Both signed and unsigned CoRIMs are supported, however signature verification
of signed CoRIMs is currently unimplemented, and so the store must be opened
with `option.Insecure` to allow signed CoRIMs to be added.

Verifiers can retrieve endorsements from the store using an Environment and the
label under which the values where added:

	env := comid.Environment{
	        Class: &comid.Class{Vendor: "ACME Inc."},
	}

	refVals, err := repo.GetActiveValueTriples(&env, "mylabel", true)
	if err != nil {
	    return err
	}

The final boolean argument indicates whether the Environment will be matched
exactly (parameters unset in the environment must also be unset in the store).
Only triples that active and within their validity period (for those that has
it) will be returned. There an analogous `GetActiveKeyTriples()` to retrieve
trust anchors.

`GetActiveKeyTriples()` and `GetActiveValueTriples()` are convenience methods
specifically for verifiers. They are implemented on top of a general query
interface.

Contents of the store may be queried via one of the query methods using the
corresponding query object.

	query := store.NewValueTripleQuery().
	    Label("mylabel").
	    ModuleTagVerision(2).
	    Environment(func(e *store.EnvironmentQuery) {
	            e.Vendor("ACME Inc.").
	                Model("Road Runner Trap 3000")
	    }).
	    ValidOn(time.Now())

	triples, err := o.QueryValueTriples(query)
	if err != nil {
	    return err
	}

Unlike the Get methods, the Query methods do not make assumptions about what is
wanted; e.g. if you're only interested in active triples, you need to specify
this as part of the query.

A CoSERV service wrapper for the Store allows running CoSERV queries and
generating coserv.ResultSet's:

	name := "Road Runner Trap 3000"
	expiration := time.Duration(10 * 24 * float64(time.Hour)) // 10 days
	authority, err := comid.NewCryptoKeyTaggedBytes("test")
	if err != nil {
	    return err
	}

	service := NewCoSERVService(store, expiration, authority)

	cs = &coserv.Coserv{
	    Profile: *profile,
	    Query: coserv.Query{
	        ArtifactType: coserv.ArtifactTypeReferenceValues,
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
	        Timestamp:  time.Now(),
	        ResultType: coserv.ResultTypeCollectedArtifacts,
	    },
	}

	// Populate cs.ResultSet with query results
	if err = service.UpdateCoSERV(cs); err != nil {
	    return err
	}
*/
package store
