package brokerintegration_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"net/http"
)

var _ = Describe("Basic HTTP Authentication", func() {

	Context("With expected username and password", func() {
		It("returns HTTP code 200", func() {
			code, _ := executeAuthenticatedHTTPRequest("GET", "http://localhost:3000/v2/catalog")
			Ω(code).To(Equal(200))
		})
	})

	Context("With unexpected username and password", func() {
		It("returns HTTP code 401", func() {
			client := &http.Client{}
			resp, err := client.Get("http://localhost:3000")
			defer resp.Body.Close()
			Ω(err).ToNot(HaveOccurred())
			req, err := http.NewRequest("GET", "http://localhost:3000/v2/catalog", nil)
			Ω(err).ToNot(HaveOccurred())
			req.SetBasicAuth("admin", "badpassword")
			resp, err = client.Do(req)
			Ω(err).ToNot(HaveOccurred())

			Ω(resp.StatusCode).To(Equal(401))
		})
	})

})
