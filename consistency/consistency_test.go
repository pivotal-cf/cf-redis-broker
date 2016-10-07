package consistency_test

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
	. "github.com/st3v/glager"

	"github.com/pivotal-cf/cf-redis-broker/consistency"
	"github.com/pivotal-cf/cf-redis-broker/redis"
)

var _ = Describe("consistency", func() {
	Describe(".KeepVerifying", func() {
		var (
			server        *ghttp.Server
			logger        *TestLogger
			statefilePath string
			keycount      int

			interval = 100 * time.Millisecond
		)

		JustBeforeEach(func() {
			server = ghttp.NewServer()
			server.RouteToHandler("GET", "/keycount",
				ghttp.RespondWith(http.StatusOK, fmt.Sprintf(`{"key_count": %d}`, keycount)),
			)

			host, port, err := net.SplitHostPort(server.Addr())
			Expect(err).ToNot(HaveOccurred())

			agentClient := redis.NewRemoteAgentClient(port, "", "", false)

			logger = NewLogger("test")

			statefile, err := ioutil.TempFile("", "test")
			Expect(err).ToNot(HaveOccurred())

			state := &redis.Statefile{
				AvailableInstances: []*redis.Instance{
					&redis.Instance{ID: "instance-1", Host: host},
					&redis.Instance{ID: "instance-2", Host: host},
				},
				AllocatedInstances: []*redis.Instance{
					&redis.Instance{ID: "instance-3", Host: host},
				},
				InstanceBindings: map[string][]string{},
			}

			err = json.NewEncoder(statefile).Encode(state)
			Expect(err).ToNot(HaveOccurred())

			statefilePath = statefile.Name()

			consistency.KeepVerifying(
				agentClient,
				statefilePath,
				interval,
				logger,
			)
		})

		AfterEach(func() {
			consistency.StopVerifying()
			server.Close()
			os.Remove(statefilePath)
		})

		It("logs a start message", func() {
			Eventually(logger).Should(HaveLogged(Info(
				Action("test.consistency.keep-verifying"),
				Data("message", "started"),
			)))
		})

		Context("when available instances do not have data", func() {
			It("does not log inconsistency errors", func() {
				Consistently(logger).ShouldNot(HaveLogged(Error(AnyErr)))
			})
		})

		Context("when available instances do have data", func() {
			BeforeEach(func() {
				keycount = 1
			})

			It("does log inconsistency errors", func() {
				Eventually(logger).Should(HaveLogged(
					Error(
						consistency.ErrHasData,
						Data("instance_id", "instance-1"),
					),
					Error(
						consistency.ErrHasData,
						Data("instance_id", "instance-2"),
					)))
			})
		})
	})
})
