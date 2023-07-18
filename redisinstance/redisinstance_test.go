package redisinstance_test

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"

	"github.com/pivotal-cf/cf-redis-broker/redisinstance"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type fakeInstanceFinder struct{}

func (finder fakeInstanceFinder) IDForHost(host string) string {
	return map[string]string{
		"1.2.3.4": "1_2_3_4",
		"9.8.7.6": "9_8_7_6",
	}[host]
}

var _ = Describe("Redisinstance", func() {
	var (
		recorder *httptest.ResponseRecorder
		handler  http.HandlerFunc
	)

	BeforeEach(func() {
		recorder = httptest.NewRecorder()
		handler = redisinstance.NewHandler(fakeInstanceFinder{})
	})

	It("it responds with a 200", func() {
		request, err := http.NewRequest("GET", "http://localhost/instances?host=1.2.3.4", nil)
		Expect(err).NotTo(HaveOccurred())
		handler.ServeHTTP(recorder, request)

		Expect(recorder.Code).To(Equal(http.StatusOK))
	})

	It("returns the correct instance id for the host provided", func() {
		request, err := http.NewRequest("GET", "http://localhost/instances?host=1.2.3.4", nil)
		Expect(err).NotTo(HaveOccurred())
		handler.ServeHTTP(recorder, request)

		Expect(readInstanceIDFrom(recorder.Body)).To(Equal("1_2_3_4"))
	})

	It("returns a not found in case the host is not allocated", func() {
		request, err := http.NewRequest("GET", "http://localhost/instances?host=unknown.host", nil)
		Expect(err).NotTo(HaveOccurred())
		handler.ServeHTTP(recorder, request)

		Expect(recorder.Code).To(Equal(http.StatusNotFound))

		bytes, err := ioutil.ReadAll(recorder.Body)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(bytes)).To(Equal("\n"))
	})

	Context("when host query param is not provided", func() {
		It("returns a 400", func() {
			request, err := http.NewRequest("GET", "http://localhost/instances", nil)
			Expect(err).NotTo(HaveOccurred())
			handler.ServeHTTP(recorder, request)

			Expect(recorder.Code).To(Equal(http.StatusBadRequest))
		})
	})
})

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
