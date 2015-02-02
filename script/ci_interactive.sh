#!/usr/bin/env bash

set -e

echo "** TESTING Redis broker"

docker logout

docker run -i -t \
  -e "GOPATH=/Users/pivotal/go" \
  -v /Users/pivotal/go:/Users/pivotal/go \
  -w /Users/pivotal/go/src/github.com/pivotal-cf/cf-redis-broker \
  cflondonservices/london-services-ci-redis:stable bash
