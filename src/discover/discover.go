package discover

// Library to return a list of hosts according to a provider
// Inspired by https://github.com/hashicorp/go-discover

type Discover interface {
	Discover() ([]string, error)
}
