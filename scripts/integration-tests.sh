#!/usr/bin/env bash

error='\e[0;31mERROR\e[0m'
this_dir=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
root_dir=$this_dir/..
corim_store=$root_dir/corim-store

function check_installed() {
        what=$1

	if [[ "$(type -p "$what")" == "" ]]; then
		echo -e "$error: $what must be installed."
		exit 1
	fi
}

function check_requirements() {
    check_installed docker
    check_installed jq
}

function setup() {
    echo "updating submods..."
    git submodule update --init

    echo "building corim-store..."
    make -s build

    if ! "$root_dir"/scripts/db-container.sh check-image; then
        echo "building DB container..."
        "$root_dir"/scripts/db-container.sh build >/dev/null
    fi

    echo "starting DB container..."
    "$root_dir"/scripts/db-container.sh start >/dev/null

    echo "initializing store..."
    $corim_store --config "$root_dir"/sample/config/sqlite3.yaml db init
    $corim_store --config "$root_dir"/sample/config/mariadb.yaml db init
    $corim_store --config "$root_dir"/sample/config/postgres.yaml db init
}

function teardown() {
    echo "stopping DB container..."
    "$root_dir"/scripts/db-container.sh stop >/dev/null

    echo "done."
}

function run() {
    "$root_dir"/test/bats/bin/bats test
}

function help() {
	set +e
	local usage
	read -r -d '' usage <<-EOF
        Usage: db-container.sh COMMAND

        Commands:

        help
            Print this message and exist (the same as -h).

        setup
            Set up integration tests infrastructure.

        run
            Run integration tests.

        teardown
            Tear down integartion tests infrastructure.
	EOF
	set -e

	echo "$usage"
}

check_requirements

while getopts "h" opt; do
	case "$opt" in
		h) help; exit 0;;
		*) break;;
	esac
done

shift $((OPTIND-1))
[ "${1:-}" = "--" ] && shift

if [[ "${1:-}" == "" ]]; then
	echo -e "$error: a command must be specified (see -h output)."
	exit 1
fi

command=${1:-}; shift
command=$(echo "$command" | tr -- _ -)
case $command in
	help) help;;
	setup) setup;;
	run ) run;;
	teardown) teardown;;
	*) echo -e "$error: unexpected command: \"$command\"";;
esac
