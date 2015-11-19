#!/bin/bash

export GOPATH=$GOPATH:$PWD
cd src/github.com/pivotal-cf/cf-redis-broker
./script/test
