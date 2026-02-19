function setup_file() {
    this_dir=$( cd -- "$( dirname -- "${BATS_TEST_FILENAME}" )" &> /dev/null && pwd )
    ROOT_DIR=$this_dir/..
    CORIM_STORE=$ROOT_DIR/corim-store

    export ROOT_DIR
    export CORIM_STORE
}

function setup() {
    load 'test_helper/bats-support/load'
    load 'test_helper/bats-assert/load'

    $CORIM_STORE --config "$ROOT_DIR"/sample/config/mariadb.yaml db migrate
    $CORIM_STORE --config "$ROOT_DIR"/sample/config/postgres.yaml db migrate
}

function teardown() {
    $CORIM_STORE --config "$ROOT_DIR"/sample/config/mariadb.yaml db rollback
    $CORIM_STORE --config "$ROOT_DIR"/sample/config/postgres.yaml db rollback
}


function do_get() {
    config_path=$1

     $CORIM_STORE --config "$config_path" corim add "$ROOT_DIR"/sample/corim/*cbor &>/dev/null
     $CORIM_STORE --config "$config_path" get --class-id psa.impl-id:f0VMRgIBAQAAAAAAAAAAAAMAPgABAAAAUFgAAAAAAAA=
}

@test "SQLite3 get" {
    run bats_pipe \
        do_get "$ROOT_DIR"/sample/config/sqlite3.yaml \| jq ".\"reference-values\" | length"
    assert_output "1"
}

@test "MariaDB get" {
    run bats_pipe \
        do_get "$ROOT_DIR"/sample/config/mariadb.yaml \| jq ".\"reference-values\" | length"
    assert_output "1"
}

@test "PostgreSQL get" {
    run bats_pipe \
        do_get "$ROOT_DIR"/sample/config/postgres.yaml \| jq ".\"reference-values\" | length"
    assert_output "1"
}
