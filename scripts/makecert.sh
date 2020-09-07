#!/bin/bash
rm -rf certs
mkdir certs
cd certs

# Extensions required for certificate validation.
EXTFILE='extfile.conf'
#echo "basicConstraints = critical,CA:true"$'\n'"subjectAltName = IP:127.0.0.1" > "${EXTFILE}"
printf 'subjectAltName = IP:127.0.0.1\nbasicConstraints = critical,CA:true' > "${EXTFILE}"

# Server.
SERVER_NAME='server'
openssl ecparam -name prime256v1 -genkey -noout -out "${SERVER_NAME}-key.pem"
openssl req -key "${SERVER_NAME}-key.pem" -new -sha256 -subj '/C=NL' -out "${SERVER_NAME}.csr"
openssl x509 -req -in "${SERVER_NAME}.csr" -extfile "${EXTFILE}" -days 365 -signkey "${SERVER_NAME}-key.pem" -sha256 -out "${SERVER_NAME}.pem"

# Client.
CLIENT_NAME='client'
openssl ecparam -name prime256v1 -genkey -noout -out "${CLIENT_NAME}-key.pem"
openssl req -key "${CLIENT_NAME}-key.pem" -new -sha256 -subj '/C=NL' -out "${CLIENT_NAME}.csr"
openssl x509 -req -in "${CLIENT_NAME}.csr" -extfile "${EXTFILE}" -days 365 -CA "${SERVER_NAME}.pem" -CAkey "${SERVER_NAME}-key.pem" -set_serial '0xabcd' -sha256 -out "${CLIENT_NAME}.pem"

# Cleanup.
rm "${EXTFILE}" "${SERVER_NAME}.csr" "${CLIENT_NAME}.csr"