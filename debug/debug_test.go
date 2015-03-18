package debug_test

import (
	"encoding/json"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"net/http"
)

var _ = Describe("Handler", func() {
	var response *http.Response

	BeforeEach(func() {
		client := http.Client{}

		var err error
		response, err = client.Get("http://localhost:3000/debug")
		Ω(err).ShouldNot(HaveOccurred())
	})

	It("returns 200", func() {
		Ω(response.StatusCode).Should(Equal(http.StatusOK))
	})

	It("returns correct content header", func() {
		Ω(response.Header.Get("Content-Type")).Should(Equal("application/json"))
	})

	Describe("Deserialize JSON response", func() {
		var debugInfo struct {
			Pool struct {
				Count    int        `json:"count"`
				Clusters [][]string `json:"clusters"`
			} `json:"pool"`
			Allocated struct {
				Count    int `json:"count"`
				Clusters []struct {
					ID       string
					Hosts    []string `json:"hosts"`
					Bindings []struct {
						ID string `json:"id"`
					} `json:"bindings"`
				} `json:"clusters"`
			} `json:"allocated"`
		}

		BeforeEach(func() {
			err := json.NewDecoder(response.Body).Decode(&debugInfo)
			Ω(err).ShouldNot(HaveOccurred())
		})

		It("has the correct pool size count", func() {
			Ω(debugInfo.Pool.Count).Should(Equal(3))
		})

		It("has the correct IP addresses of the pool", func() {
			Ω(debugInfo.Pool.Clusters).Should(ContainElement([]string{"10.0.0.1"}))
			Ω(debugInfo.Pool.Clusters).Should(ContainElement([]string{"10.0.0.2"}))
			Ω(debugInfo.Pool.Clusters).Should(ContainElement([]string{"10.0.0.3"}))
			Ω(len(debugInfo.Pool.Clusters)).Should(Equal(3))
			Ω(len(debugInfo.Allocated.Clusters)).Should(Equal(0))
		})
	})
})
