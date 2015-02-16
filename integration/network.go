package integration

import (
	"bytes"
	"io/ioutil"
	"net"
	"net/http"

	. "github.com/onsi/gomega"
)

func ExecuteAuthenticatedHTTPRequest(method, uri, username, password string) (int, []byte) {
	return ExecuteAuthenticatedHTTPRequestWithBody(method, uri, username, password, nil)
}

func ExecuteAuthenticatedHTTPRequestWithBody(method, uri, username, password string, body []byte) (int, []byte) {
	req, err := http.NewRequest(method, uri, bytes.NewReader(body))
	Ω(err).ToNot(HaveOccurred())
	req.SetBasicAuth(username, password)
	resp, err := (&http.Client{}).Do(req)
	Ω(err).ToNot(HaveOccurred())

	defer resp.Body.Close()

	responseBody, err := ioutil.ReadAll(resp.Body)
	Ω(err).ToNot(HaveOccurred())

	Ω(err).ToNot(HaveOccurred())
	return resp.StatusCode, responseBody
}

func HostIP4Addresses() ([]string, error) {
	addrs, err := net.InterfaceAddrs()

	if err != nil {
		return nil, err
	}

	ipAddresses := []string{}

	for _, address := range addrs {
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				ipAddresses = append(ipAddresses, ipnet.IP.String())
			}
		}
	}

	return ipAddresses, nil
}
