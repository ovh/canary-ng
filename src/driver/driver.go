package driver

const (
	TIMEOUT = 3
)

type Driver interface {
	Connect() error
	Read() error
	Write() error
	Disconnect() error
}
