#!/bin/bash

set -eu

test_args=$@
DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
ROOT="$( dirname $DIR )"

eval "$(gimme 1.9)"

GOPATH="$( mktemp -d )"
mkdir -p $GOPATH/src/github.com/pivotal-cf
cp -r $ROOT $GOPATH/src/github.com/pivotal-cf/

pushd $GOPATH/src/github.com/pivotal-cf/cf-redis-broker
  GOPATH=$PWD/Godeps/_workspace:$GOPATH
  ginkgo -r -race --keepGoing -randomizeAllSpecs -skipMeasurements -failOnPending -cover $test_args
popd

rm -rf $GOPATH
