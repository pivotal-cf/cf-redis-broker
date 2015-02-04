package redisconf_test

import (
	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/reporters"
	. "github.com/onsi/gomega"

	"testing"
)

func TestRedisconf(t *testing.T) {
	RegisterFailHandler(Fail)
	junitReporter := reporters.NewJUnitReporter("junit_redisconf.xml")
	RunSpecsWithDefaultAndCustomReporters(t, "Redisconf Suite", []Reporter{junitReporter})
}
