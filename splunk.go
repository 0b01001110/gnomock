// Package splunk includes Splunk Enterprise implementation of Gnomock Preset
// interface. This Preset can be passed to gnomock.StartPreset function to
// create a configured Splunk container to use in tests.
//
// Splunk image is relatively heavy (larger than 1.5GB), and its startup time
// is longer than usual. Using this container may make the tests much longer
package splunk

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/orlangure/gnomock"
)

const (
	// CollectorPort is the name of a port exposed by Splunk Collector
	CollectorPort string = "collector"

	// APIPort is the name of a port exposed by Splunk API
	APIPort string = "api"

	// WebPort is the name of a port exposed by Splunk web UI
	WebPort string = "web"
)

// Preset creates a new Gnomock Splunk preset. This preset includes a Splunk
// specific healthcheck function, default Splunk image and ports, and allows to
// optionally ingest initial logs
func Preset(opts ...Option) *Splunk {
	config := buildConfig(opts...)

	s := &Splunk{
		initialValues: config.values,
		acceptLicense: config.acceptLicense,
		adminPassword: config.adminPassword,
	}

	return s
}

// Splunk is a Gnomock Preset implementation of Splunk Enterprise
type Splunk struct {
	initialValues []Event
	acceptLicense bool
	adminPassword string
}

// Image returns an image that should be pulled to create this container
func (s *Splunk) Image() string {
	return "docker.io/splunk/splunk"
}

// Ports returns ports that should be used to access this container
func (s *Splunk) Ports() gnomock.NamedPorts {
	return gnomock.NamedPorts{
		CollectorPort: gnomock.TCP(8088),
		APIPort:       gnomock.TCP(8089),
		WebPort:       gnomock.TCP(8000),
	}
}

// Options returns a list of options to configure this container
func (s *Splunk) Options() []gnomock.Option {
	opts := []gnomock.Option{
		gnomock.WithStartTimeout(time.Minute * 5),
		gnomock.WithWaitTimeout(time.Minute * 1),
		gnomock.WithHealthCheck(healthcheck(s.adminPassword)),
		gnomock.WithEnv("SPLUNK_PASSWORD=" + s.adminPassword),
	}

	if s.acceptLicense {
		opts = append(
			opts,
			gnomock.WithEnv("SPLUNK_START_ARGS=--accept-license"),
		)
	}

	if s.initialValues != nil {
		init := initf(s.adminPassword, s.initialValues)
		opts = append(opts, gnomock.WithInit(init))
	}

	return opts
}

func healthcheck(password string) gnomock.HealthcheckFunc {
	client := insecureClient()

	return func(c *gnomock.Container) (err error) {
		addr := c.Address(APIPort)
		uri := fmt.Sprintf("https://%s/services/auth/login", addr)

		data := url.Values{}
		data.Add("username", "admin")
		data.Add("password", password)
		data.Add("output_mode", "json")

		buf := bytes.NewBufferString(data.Encode())

		resp, err := client.Post(uri, "application/x-www-form-urlencoded", buf)
		if err != nil {
			return err
		}

		defer func() {
			closeErr := resp.Body.Close()
			if err == nil && closeErr != nil {
				err = closeErr
			}
		}()

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("bad response: %s", resp.Status)
		}

		return nil
	}
}

func insecureClient() http.Client {
	return http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, //nolint:gosec
		},
	}
}
