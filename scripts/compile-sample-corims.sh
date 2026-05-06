#!/usr/bin/env bash
set -euo pipefail

CORIM_TOOL=${CORIM_TOOL:-corim-tool}

if [[ "$(type -p "${CORIM_TOOL}")" == "" ]]; then
	echo -e "\033[1;31mERROR\033[0m: corim-tool must be installed to run this script."
	echo -e "\033[1;31mERROR\033[0m: see: https://github.com/veraison/corim-tool"
	exit 1
fi

REQUIRED_VERSION="0.2.0"
ACTUAL_VERSION=$(${CORIM_TOOL} -V | cut -d" " -f2)

version_ge() {
	[ "$(printf '%s\n' "${1}" "${2}" | sort -V | head -n1)" = "${2}" ]
}

if ! version_ge "${ACTUAL_VERSION}" "${REQUIRED_VERSION}"; then
	echo -e "\033[1;31mERROR\033[0m: corim-tool version must be at least ${REQUIRED_VERSION}"
	exit 1
fi

THIS_DIR=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
SAMPLE_DIR=${THIS_DIR}/../sample/corim

for file in "${SAMPLE_DIR}"/*.json; do
	outfile=$(basename "$file")
	outfile=${outfile//corim-/unsigned-}
	signed_outfile=${outfile//unsigned-/signed-}
	signed_with_cert_outfile=${outfile//unsigned-/signed-with-cert-}
	outfile=${SAMPLE_DIR}/${outfile%.json}.cbor
	signed_outfile=${SAMPLE_DIR}/${signed_outfile%.json}.cose
	signed_with_cert_outfile=${SAMPLE_DIR}/${signed_with_cert_outfile%.json}.cose
	key="${SAMPLE_DIR}"/key.priv.pem

	echo "Compiling ${outfile}..."
	${CORIM_TOOL} compile --force "${file}" --output "${outfile}"
	${CORIM_TOOL} compile --force "${file}" --key "${key}" --output "${signed_outfile}"
	${CORIM_TOOL} compile --force "${file}" --key "${key}" --output "${signed_with_cert_outfile}" \
		--cert "$SAMPLE_DIR"/certs/leaf.cert.der \
		--cert "$SAMPLE_DIR"/certs/int.cert.der
done
