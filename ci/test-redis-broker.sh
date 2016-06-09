#!/bin/bash

export GOPATH=$PWD:$GOPATH
export RBENV_ROOT=/home/vcap/.rbenv

cd src/github.com/pivotal-cf/cf-redis-broker

# Run tests as vcap user
useradd vcap
sudo chown -R vcap:vcap /tmp

sudo -E -u vcap ./script/test
