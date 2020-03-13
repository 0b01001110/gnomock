package gnomock

import (
	"fmt"
)

// Container represents a docker container created for testing. Host and Port
// fields should be used to configure the connection to this container. ID
// matches the original docker container ID
type Container struct {
	ID    string
	Host  string
	Ports NamedPorts
}

// Address is a convenience function that returns host:port that can be used to
// connect to this container
func (c *Container) Address(name string) string {
	p := c.Ports.Get(name)
	return fmt.Sprintf("%s:%d", c.Host, p.Port)
}
