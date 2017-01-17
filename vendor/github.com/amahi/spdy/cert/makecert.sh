#!/bin/bash
# call this script with an email address (valid or not).
# like:
# ./makecert.sh joe@random.com
mkdir serverTLS
rm serverTLS/*
mkdir clientTLS
rm clientTLS/*
echo "make server cert"
openssl req -new -nodes -x509 -out serverTLS/server.pem -keyout serverTLS/server.key -days 3650 -subj "/C=DE/ST=NRW/L=Earth/O=Random Company/OU=IT/CN=www.random.com/emailAddress=$1"
echo "make client cert"
openssl req -new -nodes -x509 -out clientTLS/client.pem -keyout clientTLS/client.key -days 3650 -subj "/C=DE/ST=NRW/L=Earth/O=Random Company/OU=IT/CN=www.random.com/emailAddress=$1"
