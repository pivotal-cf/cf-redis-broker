#!/usr/bin/env bash

set -e

echo "** TESTING Redis broker"

docker logout

docker run \
  -e "GOPATH=/home/user/go" \
  -v $PWD:/home/user/go/src/github.com/pivotal-cf/cf-redis-broker \
  -w /home/user/go/src/github.com/pivotal-cf/cf-redis-broker\
  cflondonservices/london-services-ci-redis:stable ./script/test $@
