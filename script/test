#!/bin/bash
# vim: set ft=sh

set -e
set -o pipefail

test_args=$@

export PATH=$GOROOT/bin:$GOPATH/bin:$PATH

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
PROJECT_DIR="$( dirname $SCRIPT_DIR )"

__message() {
  local _message=$1
  echo -e "${_message}..."
}

main() {
  __message "Running tests"

  ginkgo -r -race --keepGoing -randomizeAllSpecs -skipMeasurements -failOnPending -cover $test_args
}

pushd $PROJECT_DIR
  main
popd
