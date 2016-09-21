package consistency_test

import (
	"encoding/json"
	"io/ioutil"
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/pivotal-cf/cf-redis-broker/consistency"
	"github.com/pivotal-cf/cf-redis-broker/redis"
)

var _ = Describe("StateFileAvailableInstances", func() {
	Describe(".Instances", func() {
		var (
			statefile          *os.File
			availableInstances []*redis.Instance
			allocatedInstances []*redis.Instance
			instancesProvider  consistency.InstancesProvider
		)

		Context("when the file does not exist", func() {
			BeforeEach(func() {
				instancesProvider = consistency.NewStateFileAvailableInstances("/i/do/not/exist.json")
			})

			It("returns an error", func() {
				_, err := instancesProvider.Instances()
				Expect(err).To(HaveOccurred())
				Expect(err).To(BeAssignableToTypeOf(new(os.PathError)))
			})
		})

		Context("when the file does exist but is not of the correct format", func() {
			BeforeEach(func() {
				var err error
				statefile, err = ioutil.TempFile("", "consistency_test")
				Expect(err).ToNot(HaveOccurred())

				data := []byte("i am not in json format")
				_, err = statefile.Write(data)
				Expect(err).ToNot(HaveOccurred())

				instancesProvider = consistency.NewStateFileAvailableInstances(statefile.Name())
			})

			AfterEach(func() {
				os.Remove(statefile.Name())
			})

			It("returns an error", func() {
				_, err := instancesProvider.Instances()
				Expect(err).To(HaveOccurred())
				Expect(err).To(BeAssignableToTypeOf(new(json.SyntaxError)))
			})
		})

		Context("when the files does exist and there are available instances", func() {
			BeforeEach(func() {
				availableInstances = []*redis.Instance{
					{ID: "1"},
					{ID: "2"},
				}

				allocatedInstances = []*redis.Instance{
					{ID: "3"},
					{ID: "4"},
				}

				state := &redis.Statefile{
					AvailableInstances: availableInstances,
					AllocatedInstances: allocatedInstances,
				}

				var err error
				statefile, err = ioutil.TempFile("", "consistency_test")
				Expect(err).ToNot(HaveOccurred())

				err = json.NewEncoder(statefile).Encode(state)
				Expect(err).ToNot(HaveOccurred())

				instancesProvider = consistency.NewStateFileAvailableInstances(statefile.Name())
			})

			AfterEach(func() {
				os.Remove(statefile.Name())
			})

			It("it returns only those instances", func() {
				instances, err := instancesProvider.Instances()

				Expect(err).NotTo(HaveOccurred())

				Expect(instances).To(HaveLen(len(availableInstances)))
				for i, instance := range instances {
					Expect(instance).To(Equal(*availableInstances[i]))
				}
			})
		})
	})
})
