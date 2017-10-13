#!/bin/bash

ps ax | grep 'redis-server .*:' | grep -v grep | wc -l
