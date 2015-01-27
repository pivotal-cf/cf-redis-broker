#!/usr/bin/env bash

set -e

echo "** TESTING Redis broker"

docker run \
  -e "GOPATH=/home/user/go" \
  -v $PWD:/home/user/go/src/github.com/pivotal-cf/cf-redis-broker \
  -w /home/user/go/src/github.com/pivotal-cf/cf-redis-broker\
  cflondonservices/london-services-ci-redis:976a703cfe079cb60047a3b3ab8d57cfc5601b06 ./script/test
