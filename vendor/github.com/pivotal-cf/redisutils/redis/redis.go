package redis

import redigo "github.com/garyburd/redigo/redis"

//Redis is an interface around redigo/redis
type Redis interface {
	Dial(string, string, ...redigo.DialOption) (redigo.Conn, error)
}

//Real is a wrapper around redigo/redis that implements redis.Redis
type Real struct{}

//New creates a struct that behaves like the redigo/redis package
func New() *Real {
	return new(Real)
}

//Dial is a wrapper around redigo/redis.Dial()
func (*Real) Dial(network, address string, options ...redigo.DialOption) (redigo.Conn, error) {
	return redigo.Dial(network, address, options...)
}
