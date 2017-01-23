package iredis

import redis "gopkg.in/redis.v5"

//Redis is an interface around redis
type Redis interface {
	NewClient(*redis.Options) Client
	NewScript(src string) Script
}

//Real is a wrapper around redis that implements iredis.Redis
type Real struct{}

//New creates a struct that behaves like the redis package
func New() *Real {
	return new(Real)
}
