#!/bin/bash

export PATH=$GOPATH/bin:$PATH
export GOPATH=$PWD:$GOPATH

cd src/github.com/pivotal-cf/cf-redis-broker

# Run tests as vcap user
useradd vcap
sudo chown -R vcap:vcap /tmp

sudo -E -u vcap ./script/test
