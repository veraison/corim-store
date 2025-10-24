#!/usr/bin/env bash
set -euo pipefail

CORIM_TOOL=${CORIM_TOOL:-corim-tool}

if [[ "$(type -p "${CORIM_TOOL}")" == "" ]]; then
	echo -e "\033[1;31mERROR\033[0m: corim-tool must be installed to run this script."
	echo -e "\033[1;31mERROR\033[0m: see: https://github.com/veraison/corim-tool"
	exit 1
fi

THIS_DIR=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
SAMPLE_DIR=${THIS_DIR}/../sample/corim

for file in "${SAMPLE_DIR}"/*.json; do
	outfile=$(basename "$file")
	outfile=${outfile//corim-/unsigned-}
	outfile=${SAMPLE_DIR}/${outfile%.json}.cbor

	echo "Compiling ${outfile}..."
	${CORIM_TOOL} compile --force "${file}" --output "${outfile}"
done
