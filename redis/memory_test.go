package redis_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/pivotal-cf/cf-redis-broker/redis"
)

var _ = Describe("Memory", func() {
	Describe("happy cases", func() {
		It("Converts", func() {
			table := map[string]string{
				"123":                 "123",
				"100kb":               "102400",
				"172Kb":               "176128",
				"999MB":               "1047527424",
				"333gb":               "357556027392",
				"1,000kb":             "1024000",
				"1,000 kb":            "1024000",
				"9223372036854775807": "9223372036854775807",
			}

			for input, output := range table {

				val, err := redis.ParseMemoryStringToBytes(input)
				Ω(err).ShouldNot(HaveOccurred())
				Ω(val).Should(Equal(output))
			}
		})
	})

	Describe("sad cases", func() {
		It("Raises an error", func() {
			errorCases := []string{
				"",
				"123bg",
				"1.5 MB",
				"45pb",
				"124mb234",
				"-10kb",
				"mb",
				"xxx",
			}

			for _, input := range errorCases {
				val, err := redis.ParseMemoryStringToBytes(input)
				Ω(err).Should(HaveOccurred())
				Ω(val).Should(Equal(""))
			}
		})
	})
})
