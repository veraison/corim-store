#!/usr/bin/env bash
set -euo pipefail

COVERAGE_THRESHOLD=$(echo "${COVERAGE_THRESHOLD:-80}" | tr -d %)

IFS=' ' read -r IGNORED_PACKAGES <<< "${IGNORE_COVERAGE:-}"
ALL_IGNORED_PACKAGES=(
	# /cmd/ sub-packages contain CLI implementation and so require some
        # additional infrastructure to be unit-tested.
        # TODO: implement unit tests for CLI commands
	github.com/veraison/corim-store/cmd/corim-store
	github.com/veraison/corim-store/cmd/corim-store/cmd
	# build sub-package contains stuff specific to the build process.
	github.com/veraison/corim-store/pkg/build
	# Migrations are executed when constructed the test DB, and so are unit-tested
	# by tests outside the sub-package.
	github.com/veraison/corim-store/pkg/migrations
        # Test helpers only; does not contain production code.
	github.com/veraison/corim-store/pkg/test
	"${IGNORED_PACKAGES[@]}"
)

GO=${GO:-go}
AWK=${AWK:-awk}
SED=${SED:-sed}

for WHAT in "$SED" "$AWK"; do
	if [[ "$(type -p "${WHAT}")" == "" ]]; then
		echo -e "\033[1;31mERROR\033[0m: ${WHAT} must be installed to run the coverage script."
		exit 1
	fi
done

THIS_DIR=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
ROOT_DIR=${THIS_DIR}/..

# shellcheck disable=2016
COVERAGE_LINES=$(${GO} test -tags=test "${ROOT_DIR}"/... -coverprofile="${ROOT_DIR}/coverage.out" |  \
	${SED} -r 's/^(ok)?\s+//' | ${AWK} '{print $1" "$(NF - 2)}' | tr -d '%')

ERRORS=false
while IFS= read -r line; do
	PACKAGE=${line%% *}
	COVERAGE=${line##* }

	if [[ " ${ALL_IGNORED_PACKAGES[*]} " =~ [[:space:]]${PACKAGE}[[:space:]] ]]; then
		echo "${PACKAGE} coverage ignored"
		continue
	fi

	if ${AWK} "BEGIN{exit !(${COVERAGE} < ${COVERAGE_THRESHOLD})}"; then
		echo -e "$PACKAGE coverage: \033[1;31m${COVERAGE}%\033[0m"
		ERRORS=true
	else
		echo -e "$PACKAGE coverage: \033[1;37m${COVERAGE}%\033[0m"
	fi
done <<< "$COVERAGE_LINES"

if [[ $ERRORS == true ]]; then
	exit 1
else
	exit 0
fi
