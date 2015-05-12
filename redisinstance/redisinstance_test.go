package redisinstance_test

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"

	"github.com/pivotal-cf/cf-redis-broker/redisinstance"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

type fakeInstanceFinder struct{}

func (finder fakeInstanceFinder) IDForHost(host string) string {
	return map[string]string{
		"1.2.3.4": "1_2_3_4",
		"9.8.7.6": "9_8_7_6",
	}[host]
}

type fakeIsAllocatedChecker struct{}

func (fakeIsAllocatedChecker) IsAllocated(host string) bool {
	if host == "1.2.3.4" {
		return true
	}
	return false
}

var _ = Describe("Redisinstance", func() {
	var recorder *httptest.ResponseRecorder

	BeforeEach(func() {
		recorder = httptest.NewRecorder()
	})

	Context("Instance Finder", func() {
		It("it responds with a 200", func() {
			handler := redisinstance.NewHandler(fakeInstanceFinder{})

			request, err := http.NewRequest("GET", "http://localhost/instances?host=1.2.3.4", nil)
			Expect(err).NotTo(HaveOccurred())
			handler.ServeHTTP(recorder, request)

			Expect(recorder.Code).To(Equal(http.StatusOK))
		})

		It("returns the correct instance id for the host provided", func() {
			handler := redisinstance.NewHandler(fakeInstanceFinder{})

			request, err := http.NewRequest("GET", "http://localhost/instances?host=1.2.3.4", nil)
			Expect(err).NotTo(HaveOccurred())
			handler.ServeHTTP(recorder, request)

			Expect(readInstanceIDFrom(recorder.Body)).To(Equal("1_2_3_4"))
		})

		It("returns a not found in case the host is not allocated", func() {
			handler := redisinstance.NewHandler(fakeInstanceFinder{})

			request, err := http.NewRequest("GET", "http://localhost/instances?host=unknown.host", nil)
			Expect(err).NotTo(HaveOccurred())
			handler.ServeHTTP(recorder, request)

			Expect(recorder.Code).To(Equal(http.StatusNotFound))

			bytes, err := ioutil.ReadAll(recorder.Body)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(bytes)).To(Equal("\n"))
		})
	})

	Context("Is Allocated Finder", func() {
		It("it responds with a 200", func() {
			handler := redisinstance.NewIsAllocatedHandler(fakeIsAllocatedChecker{})

			request, err := http.NewRequest("GET", "http://localhost/is_allocated?host=1.2.3.4", nil)
			Expect(err).NotTo(HaveOccurred())
			handler.ServeHTTP(recorder, request)

			Expect(recorder.Code).To(Equal(http.StatusOK))
		})

		It("it responds with true if instance is allocated", func() {
			handler := redisinstance.NewIsAllocatedHandler(fakeIsAllocatedChecker{})

			request, err := http.NewRequest("GET", "http://localhost/is_allocated?host=1.2.3.4", nil)
			Expect(err).NotTo(HaveOccurred())
			handler.ServeHTTP(recorder, request)

			Expect(readIsAllocated(recorder.Body)).To(BeTrue())
		})

		It("it responds with false if instance is not allocated", func() {
			handler := redisinstance.NewIsAllocatedHandler(fakeIsAllocatedChecker{})

			request, err := http.NewRequest("GET", "http://localhost/is_allocated?host=unknown_host", nil)
			Expect(err).NotTo(HaveOccurred())
			handler.ServeHTTP(recorder, request)

			Expect(readIsAllocated(recorder.Body)).To(BeFalse())
		})
	})
})

func readIsAllocated(body *bytes.Buffer) bool {
	parsedBody := struct {
		IsAllocated bool `json:"is_allocated"`
	}{}

	bytes, err := ioutil.ReadAll(body)
	Expect(err).NotTo(HaveOccurred())
	err = json.Unmarshal(bytes, &parsedBody)
	Expect(err).ToNot(HaveOccurred())

	return parsedBody.IsAllocated
}

func readInstanceIDFrom(body *bytes.Buffer) string {
	parsedBody := struct {
		InstanceID string `json:"instance_id"`
	}{}

	bytes, err := ioutil.ReadAll(body)
	Expect(err).NotTo(HaveOccurred())
	err = json.Unmarshal(bytes, &parsedBody)
	Expect(err).ToNot(HaveOccurred())

	return parsedBody.InstanceID
}
