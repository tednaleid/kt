#! /usr/bin/env bash

# generate certs using instructions from https://docs.confluent.io/2.0.0/kafka/ssl.html

set -euxo pipefail

BIN_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CERT_DIR="${BIN_DIR}/certs"

SSL_PASSWORD=changeit

SERVER_KEYSTORE_JKS="$CERT_DIR/kafka.server.keystore.jks"
SERVER_TRUSTSTORE_JKS="$CERT_DIR/kafka.server.truststore.jks"  # used by client apps
CLIENT_TRUSTSTORE_JKS="$CERT_DIR/kafka.client.truststore.jks"  # used by the kafka server for which clients to trust

CA_KEY="$CERT_DIR/ca.key"
CA_CERT="$CERT_DIR/ca.cert"
CERT_FILE="$CERT_DIR/cert-file"
CERT_SIGNED="$CERT_DIR/cert-signed"

echo "removing old certs"

mkdir -p $CERT_DIR
rm -rf $CERT_DIR/*

keytool -keystore $SERVER_KEYSTORE_JKS \
        -alias localhost -validity 365 -genkey -storepass $SSL_PASSWORD -keypass $SSL_PASSWORD \
        -dname "CN=localhost, OU=None, O=None, L=Minneapolis, ST=MN, C=US"

# create CA
openssl req -new -x509 -keyout $CA_KEY -out $CA_CERT -days 365 -passout pass:$SSL_PASSWORD \
   -subj "/C=US/ST=MN/L=Minneapolis/O=None/OU=None/CN=localhost"

# add CA to truststores
keytool -keystore $SERVER_TRUSTSTORE_JKS -alias CARoot -import -file $CA_CERT -storepass $SSL_PASSWORD -noprompt

keytool -keystore $CLIENT_TRUSTSTORE_JKS -alias CARoot -import -file $CA_CERT -storepass $SSL_PASSWORD -noprompt

# export the cert so it can be signed
keytool -keystore $SERVER_KEYSTORE_JKS -alias localhost -certreq -file $CERT_FILE -storepass $SSL_PASSWORD -noprompt

# sign it with the CA
openssl x509 -req -CA $CA_CERT -CAkey $CA_KEY -in $CERT_FILE -out $CERT_SIGNED -days 365 -CAcreateserial -passin pass:$SSL_PASSWORD


# import the certificate and the CA into the keystore

keytool -keystore $SERVER_KEYSTORE_JKS -alias CARoot -import -file $CA_CERT -storepass $SSL_PASSWORD -noprompt
keytool -keystore $SERVER_KEYSTORE_JKS -alias localhost -import -file $CERT_SIGNED -storepass $SSL_PASSWORD -noprompt


# create pem file for kt

keytool -importkeystore -srckeystore $SERVER_KEYSTORE_JKS -srcstoretype JKS -deststoretype PKCS12 -destkeystore keystore.p12

ls $CERT_DIR


echo after "docker-compose up" you can test the connectivity with
echo openssl s_client -debug -connect localhost:9093 -tls1


# TODO start at configuring kafka clients on https://docs.confluent.io/2.0.0/kafka/ssl.html

