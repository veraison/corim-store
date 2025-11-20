#!/usr/bin/env bash
set -ueo pipefail

TEST_DB_CONTAINER_NAME=${TEST_DB_CONTAINER_NAME:-corim-store-db}
TEST_DB_MYSQL_PORT=${TEST_DB_MYSQL_PORT:-33306}
TEST_DB_POSTGRES_PORT=${TEST_DB_POSTGRES_PORT:-55432}
TEST_DB_LOG_FILE=${TEST_DB_LOG_FILE:-/tmp/corim-store-db-output.log}

error='\e[0;31mERROR\e[0m'
this_dir=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
root_dir=$this_dir/..
mysql_args="--ssl=0 -h 127.0.0.1 -P $TEST_DB_MYSQL_PORT -u store_user --password=L3tM31n corim_store"
psql_args="-U store_user -h localhost -p 55432 -d corim_store"

function check_image() {
    if [[ "$(docker images -q "$TEST_DB_CONTAINER_NAME")" == "" ]]; then
	    exit 1
    fi
}

function build() {
	if [[ "$(type -p docker)" == "" ]]; then
		echo -e "$error: docker must be installed."
		exit 1
	fi

	if ! (docker buildx version &>/dev/null); then
		echo -e "$error: docker-buildx Docker plugin must be installed."
		exit 1
	fi

	pushd "$root_dir/docker" &>/dev/null
	trap 'popd &>/dev/null' RETURN

	echo "building $TEST_DB_CONTAINER_NAME..."
	docker build . -t "$TEST_DB_CONTAINER_NAME"
	echo "done."
}

function run() {
	if [[ "$(docker images -q "$TEST_DB_CONTAINER_NAME")" == "" ]]; then
		echo -e "$error: container $TEST_DB_CONTAINER_NAME does not exist " \
			"(run db-container.sh build first)."
		exit 1
	fi

	echo "running $TEST_DB_CONTAINER_NAME..."
	echo "----------------------------------------------------" >>"$TEST_DB_LOG_FILE"
	nohup docker run -p "${TEST_DB_MYSQL_PORT}":3306 -p "${TEST_DB_POSTGRES_PORT}":5432 \
		"$TEST_DB_CONTAINER_NAME" &>>"$TEST_DB_LOG_FILE" &

	if [[ "$(type -p mariadb)" != "" ]]; then
		sleep 2
		count=0
		# shellcheck disable=SC2086
		until mariadb $mysql_args -e "SELECT 1" &>/dev/null; do
			sleep 1
			((count += 1))

			if [[ $count -gt 20 ]]; then
				echo -e "$error: timed out waiting for MariaDB server."
				exit 1
			fi
		done
	else
		# don't have MariaDB client not installed on the host, so just use a constant delay
		# to give the server time to start.
		sleep 10
	fi

	echo "done."
}

function stop() {
	echo "stopping $TEST_DB_CONTAINER_NAME..."
	for cid in $(docker ps -q --filter "ancestor=$TEST_DB_CONTAINER_NAME"); do
		docker stop "$cid"
	done
	echo "done."
}

function shell() {
	for cid in $(docker ps -q --filter "ancestor=$TEST_DB_CONTAINER_NAME"); do
		docker exec -it "$cid" /bin/bash
		break
	done
}

function mariadb_shell() {
	if [[ "$(type -p mariadb)" == "" ]]; then
		echo -e "$error: MariaDB client (mariadb) must be installed."
		exit 1
	fi

	echo "mariadb $mysql_args"
	# shellcheck disable=SC2086
	mariadb $mysql_args
}

function psql_shell() {
	if [[ "$(type -p psql)" == "" ]]; then
		echo -e "$error: PosogreSQL client (psql) must be installed."
		exit 1
	fi

	export PGPASSWORD='L3tM31n'
	echo "PGPASSWORD=$PGPASSWORD psql $psql_args"
	# shellcheck disable=SC2086
	psql $psql_args
}

function help() {
	set +e
	local usage
	read -r -d '' usage <<-EOF
	Usage: db-container.sh COMMAND

	Commands:

	help
	    Print this message and exist (the same as -h).

	check-image
	    Check whether the Docker container image has been created. Non-zero exit code
	    indicates that it was not.

	build
	    Build Docker image that runs MariaDB and PosogreSQL servers with appropriate
	    database and users created.

	run | start
	    Run a Docker container built using the build command (see above).

	stop
	    Stop the container started with run/start command.

	shell
	    Start a (root) shell  inside the container.

	mariadb | mysql
	   Start an interactive session shell on the MariaDB server running inside the container.

	postgres | psql
	   Start an interactive session shell on the PostgreSQL server running inside the container.

	Environment Variables:

	A number of environment variables control the Docker image/container. Please make sure
	the values of these variables remain consisten across command invocations.

	TEST_DB_CONTAINER_NAME
	    The name of the Docker container image  name that will be built. (Defaults to
	    corim-store-db).

	TEST_DB_MYSQL_PORT
	    The host post that the MariaDB servier will be listening on. (Defaults to 33306).

	TEST_DB_POSTGRES_PORT
	    The host post that the PostgreSQL servier will be listening on. (Defaults to 55432).

	TEST_DB_LOG_FILE
	   The file that the output from the running container will be logged to (Defaults to
	   /tmp/corim-store-db-output.log).

	EOF
	set -e

	echo "$usage"
}

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
	check-image) check_image;;
	build) build;;
	run | start) run;;
	mariadb | mysql) mariadb_shell;;
	postgres | psql) psql_shell;;
	shell) shell;;
	stop) stop;;
    	*) echo -e "$error: unexpected command: \"$command\"";;
esac
