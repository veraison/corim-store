#!/usr/bin/env bash
set -euo pipefail

OPENSSL=${OPENSSL:-openssl}

if [[ "$(type -p "${OPENSSL}")" == "" ]]; then
	echo -e "\033[1;31mERROR\033[0m: openssl must be installed to run this script."
	exit 1
fi


THIS_DIR=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
SAMPLE_DIR=$(realpath "${THIS_DIR}"/../sample/corim)
CERT_DIR=${SAMPLE_DIR}/certs

set -x

# root key and certificate
${OPENSSL} ecparam -name prime256v1 -genkey -noout -out "${CERT_DIR}"/root.key.pem

${OPENSSL} req -new -x509 \
	-key "${CERT_DIR}"/root.key.pem \
	-out "${CERT_DIR}"/root.cert.pem \
	-days 10000 \
	-subj "/CN=Root CA" \
	-addext  "basicConstraints=critical,CA:TRUE" \
	-addext "keyUsage=critical,keyCertSign,cRLSign" \
	-addext "subjectKeyIdentifier=hash"

# intermediate certifcate
${OPENSSL} ecparam -name prime256v1 -genkey -noout -out "${CERT_DIR}"/int.key.pem

${OPENSSL} req -new \
	-key "${CERT_DIR}"/int.key.pem \
	-out "${CERT_DIR}"/int.csr.pem \
	-subj "/CN=Intermediate CA"

${OPENSSL} x509 -req \
	-in "${CERT_DIR}"/int.csr.pem \
	-CA "${CERT_DIR}"/root.cert.pem \
	-CAkey "${CERT_DIR}"/root.key.pem \
	-CAcreateserial \
	-out "${CERT_DIR}"/int.cert.pem \
	-days 10000 \
	-extfile "${CERT_DIR}"/v3ext.int.cnf \
	-extensions v3_ca

# leaf certificate
${OPENSSL} req -new \
	-key "${SAMPLE_DIR}"/key.priv.pem \
	-out "${CERT_DIR}"/leaf.csr.pem \
	-subj "/CN=CoRIM Signer"

${OPENSSL} x509 -req \
	-in "${CERT_DIR}"/leaf.csr.pem \
	-CA "${CERT_DIR}"/int.cert.pem \
	-CAkey "${CERT_DIR}"/int.key.pem \
	-CAcreateserial \
	-out "${CERT_DIR}"/leaf.cert.pem \
	-days 10000 \
	-extfile "${CERT_DIR}"/v3ext.leaf.cnf \
	-extensions v3_leaf

# PEM -> DER
${OPENSSL} x509 -in "${CERT_DIR}"/root.cert.pem -outform DER -out "${CERT_DIR}"/root.cert.der
${OPENSSL} x509 -in "${CERT_DIR}"/int.cert.pem -outform DER -out "${CERT_DIR}"/int.cert.der
${OPENSSL} x509 -in "${CERT_DIR}"/leaf.cert.pem -outform DER -out "${CERT_DIR}"/leaf.cert.der
