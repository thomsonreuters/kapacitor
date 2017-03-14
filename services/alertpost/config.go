package alertpost

import (
	"io"
	"net/http"
	"net/url"

	"github.com/pkg/errors"
)

// Config is the configuration for a single [[alertpost]] section of the kapacitor
// configuration file.
type Config struct {
	Endpoint string            `toml:"endpoint" override:"endpoint"`
	URL      string            `toml:"url" override:"url"`
	Headers  map[string]string `toml:"headers" override:"headers,redact"`
}

// NewRequest generates wraps a call to http.NewRequest that sets the headers
// for a HTTP request as defined by the configuration.
func (c Config) NewRequest(body io.Reader) (*http.Request, error) {
	req, err := http.NewRequest("POST", c.URL, body)
	if err != nil {
		return nil, err
	}

	for k, v := range c.Headers {
		req.Header.Add(k, v)
	}

	return req, nil
}

// Validate ensures that all configurations options are valid. The Endpoint,
// and URL parameters must be set to be considered valid.
func (c Config) Validate() error {
	if c.Endpoint == "" {
		return errors.New("must specify endpoint name")
	}

	if c.URL == "" {
		return errors.New("must specify url")
	}

	if _, err := url.Parse(c.URL); err != nil {
		return errors.Wrapf(err, "invalid URL %q", c.URL)
	}

	return nil
}

// Configs is the configuration for all [[alertpost]] sections of the kapacitor
// configuration file.
type Configs []Config

// Validate calls config.Validate for each element in Configs
func (cs Configs) Validate() error {
	for _, c := range cs {
		err := c.Validate()
		if err != nil {
			return err
		}
	}
	return nil
}

// index generates a map from config.Endpoint to config
func (cs Configs) index() map[string]Config {
	m := map[string]Config{}

	for _, c := range cs {
		m[c.Endpoint] = c
	}

	return m
}
