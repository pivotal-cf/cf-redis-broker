#!/usr/bin/env bash

set -e

echo "** TESTING Redis broker"

docker logout

docker run \
  -e "GOPATH=/home/user/go" \
  -e "AWS_ACCESS_KEY_ID=${AWS_ACCESS_KEY_ID}" \
  -e "AWS_SECRET_ACCESS_KEY=${AWS_SECRET_ACCESS_KEY}" \
  -v $PWD:/home/user/go/src/github.com/pivotal-cf/cf-redis-broker \
  -w /home/user/go/src/github.com/pivotal-cf/cf-redis-broker\
  cflondonservices/london-services-ci-redis:stable ./script/test $@
