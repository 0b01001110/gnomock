//nolint:gosec
package gnomock_test

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"testing"
	"time"

	"github.com/orlangure/gnomock"
	"github.com/stretchr/testify/require"
)

const testImage = "docker.io/orlangure/gnomock-test-image"
const goodPort80 = 80
const goodPort8080 = 8080
const badPort = 8000

func TestGnomock_happyFlow(t *testing.T) {
	t.Parallel()

	namedPorts := gnomock.NamedPorts{
		"web80":   gnomock.TCP(goodPort80),
		"web8080": gnomock.TCP(goodPort8080),
	}
	container, err := gnomock.Start(
		testImage, namedPorts,
		gnomock.WithHealthCheckInterval(time.Microsecond*500),
		gnomock.WithHealthCheck(healthcheck),
		gnomock.WithInit(initf),
		gnomock.WithContext(context.Background()),
		gnomock.WithStartTimeout(time.Second*30),
		gnomock.WithWaitTimeout(time.Second*1),
		gnomock.WithEnv("GNOMOCK_TEST_1=foo"),
		gnomock.WithEnv("GNOMOCK_TEST_2=bar"),
	)

	defer func() {
		require.NoError(t, gnomock.Stop(container))
	}()

	require.NoError(t, err)
	require.NotNil(t, container)

	addr := fmt.Sprintf("http://%s/", container.Address("web80"))
	requireResponse(t, addr, "80")

	addr = fmt.Sprintf("http://%s/", container.Address("web8080"))
	requireResponse(t, addr, "8080")
}

func TestGnomock_wrongPort(t *testing.T) {
	t.Parallel()

	container, err := gnomock.Start(
		testImage, gnomock.DefaultTCP(badPort),
		gnomock.WithHealthCheck(healthcheck),
		gnomock.WithWaitTimeout(time.Millisecond*50),
	)

	defer func() {
		require.NoError(t, gnomock.Stop(container))
	}()

	require.Error(t, err)
	require.NotNil(t, container)
}

func TestGnomock_cancellation(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		time.Sleep(time.Millisecond * 100)
		cancel()
	}()

	container, err := gnomock.Start(
		testImage, gnomock.DefaultTCP(badPort),
		gnomock.WithHealthCheck(healthcheck),
		gnomock.WithContext(ctx),
	)

	defer func() {
		require.NoError(t, gnomock.Stop(container))
	}()

	require.True(t, errors.Is(err, context.Canceled))
}

func TestGnomock_defaultHealthcheck(t *testing.T) {
	t.Parallel()

	container, err := gnomock.Start(testImage, gnomock.DefaultTCP(badPort))

	defer func() {
		require.NoError(t, gnomock.Stop(container))
	}()

	// there is no error since healthcheck never returns an error
	require.NoError(t, err)
}

func TestGnomock_initError(t *testing.T) {
	t.Parallel()

	errNope := fmt.Errorf("nope")
	initWithErr := func(*gnomock.Container) error {
		return errNope
	}

	container, err := gnomock.Start(
		testImage, gnomock.DefaultTCP(goodPort80),
		gnomock.WithInit(initWithErr),
	)

	defer func() {
		require.NoError(t, gnomock.Stop(container))
	}()

	require.True(t, errors.Is(err, errNope))
}

func healthcheck(c *gnomock.Container) error {
	err := callRoot(fmt.Sprintf("http://%s/", c.Address("web80")))
	if err != nil {
		return err
	}

	err = callRoot(fmt.Sprintf("http://%s/", c.Address("web8080")))
	if err != nil {
		return err
	}

	return nil
}

func callRoot(addr string) error {
	resp, err := http.Get(addr)
	if err != nil {
		return fmt.Errorf("can't GET %s: %w", addr, err)
	}

	defer func() {
		closeErr := resp.Body.Close()

		if err == nil {
			err = closeErr
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("invalid status code: %d", resp.StatusCode)
	}

	return nil
}

func initf(*gnomock.Container) error {
	return nil
}

func requireResponse(t *testing.T, url string, expected string) {
	resp, err := http.Get(url)
	require.NoError(t, err)

	defer func() {
		require.NoError(t, resp.Body.Close())
	}()

	require.Equal(t, http.StatusOK, resp.StatusCode)

	body, err := ioutil.ReadAll(resp.Body)

	require.NoError(t, err)
	require.Equal(t, expected, string(body))
}
