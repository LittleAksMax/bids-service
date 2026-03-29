package message_queue

import "fmt"

type Config struct {
	Host     string
	Port     int
	User     string
	Password string
	Queue    string
}

func (c *Config) getConnectionURL() string {
	return fmt.Sprintf("amqp://%s:%s@%s:%d/",
		c.User, c.Password, c.Host, c.Port)
}
