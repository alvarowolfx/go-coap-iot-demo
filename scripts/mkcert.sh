#!/bin/bash
rm -rf certs
mkdir certs
cd certs

mkcert --cert-file server.pem --ecdsa --key-file server-key.pem localhost 127.0.0.1 ::1
mkcert --client --ecdsa localhost 127.0.0.1 ::1
mv localhost+2-client.pem client.pem
mv localhost+2-client-key.pem client-key.pem