package latency

import (
	"fmt"
	"io/ioutil"
	"os"
	"sync"
	"time"

	"github.com/cloudfoundry-incubator/candiedyaml"
	"github.com/pivotal-cf/cf-redis-broker/redis/client"
	"github.com/pivotal-golang/lager"
)

type Latency struct {
	redisClient       client.Client
	latencyFilePath   string
	interval          time.Duration
	pingStopChan      chan (bool)
	fileWriteStopChan chan (bool)
	logger            lager.Logger
}

func NewLatency(
	redisClient client.Client,
	latencyFilePath string,
	interval time.Duration,
	logger lager.Logger,
) *Latency {
	latency := &Latency{
		redisClient:     redisClient,
		latencyFilePath: latencyFilePath,
		interval:        interval,
		logger:          logger,
	}
	latency.pingStopChan = make(chan bool)
	latency.fileWriteStopChan = make(chan bool)
	return latency
}

type Config struct {
	Interval        string `yaml:"interval"`
	LatencyFilePath string `yaml:"latency_file_path"`
}

func LoadConfig(filePath string) (*Config, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}

	config := &Config{}
	if err := candiedyaml.NewDecoder(file).Decode(config); err != nil {
		return nil, err
	}

	return config, nil
}

func (l *Latency) Start() error {
	var (
		totalDuration time.Duration
		count         int
		updateMutex   sync.Mutex
	)

	l.logger.Info("Start latency monitoring")
	go func() {
		for {
			select {
			case <-time.After(time.Millisecond * 10):
				start := time.Now()
				l.redisClient.Ping()
				duration := time.Since(start)

				func() {
					updateMutex.Lock()
					defer updateMutex.Unlock()

					totalDuration = totalDuration + duration
					count++
				}()
			case <-l.pingStopChan:
				return
			}
		}
	}()

	go func() {
		for {
			select {
			case <-time.After(l.interval):
				func() {
					updateMutex.Lock()
					defer updateMutex.Unlock()

					microTime := float64(totalDuration.Nanoseconds()/int64(count)) / 1000000
					stringDuration := fmt.Sprintf("%.2f", microTime)

					l.logger.Info("Writing latency to file", lager.Data{"Latency": stringDuration})
					ioutil.WriteFile(l.latencyFilePath, []byte(stringDuration), 0644)

					totalDuration = 0.
					count = 0
				}()

			case <-l.fileWriteStopChan:
				return
			}
		}
	}()

	return nil
}

func (l *Latency) Stop() error {
	defer close(l.pingStopChan)
	defer close(l.fileWriteStopChan)

	l.pingStopChan <- true
	l.fileWriteStopChan <- true
	return nil
}
