#!/bin/bash
RUNTIME_DIR="./.runtime"
echo "generating ecdsa key pair..."
if [ -d "$RUNTIME_DIR" ]; then rm -Rf $RUNTIME_DIR; fi
mkdir "$RUNTIME_DIR"
openssl ecparam -out "$RUNTIME_DIR/ecdsa_private.pem" -name prime256v1 -genkey
openssl ec -in "$RUNTIME_DIR/ecdsa_private.pem" -pubout -out "$RUNTIME_DIR/ecdsa_public.pem"
