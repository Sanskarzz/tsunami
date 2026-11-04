#!/bin/sh
# Original Tsunami adapter. Target URL/account are JSON data on stdin, never
# shell source or curl options. This curl talks only to the local Unix bridge.
set -eu
test "$#" -eq 1
exec /usr/bin/curl --disable --silent --show-error --fail-with-body \
  --proxy '' --noproxy '*' --proto '=http' --proto-redir '=http' \
  --max-redirs 0 --retry 0 --connect-timeout 2 --max-time 12 \
  --request POST --header 'Content-Type: application/json' \
  --unix-socket "$1" --data-binary @- http://localhost/request
