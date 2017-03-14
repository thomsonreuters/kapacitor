package alertpost

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/influxdata/kapacitor/alert"
	"github.com/influxdata/kapacitor/bufpool"
	"github.com/influxdata/kapacitor/models"
)

type Service struct {
	mu        sync.RWMutex
	endpoints map[string]Config
	logger    *log.Logger
}

func NewService(c Configs, l *log.Logger) *Service {
	s := &Service{
		logger:    l,
		endpoints: c.index(),
	}
	return s
}

type HandlerConfig struct {
	URL      string `mapstructure:"url"`
	Endpoint string `mapstructure:"endpoint"`
}

type handler struct {
	s        *Service
	bp       *bufpool.Pool
	url      string
	endpoint string
	logger   *log.Logger
}

func (s *Service) Handler(c HandlerConfig, l *log.Logger) alert.Handler {
	return &handler{
		s:        s,
		bp:       bufpool.New(),
		url:      c.URL,
		endpoint: c.Endpoint,
		logger:   l,
	}
}

func (s *Service) Open() error {
	return nil
}

func (s *Service) Close() error {
	return nil
}

func (s *Service) endpoint(e string) (c Config, ok bool) {
	s.mu.RLock()
	c, ok = s.endpoints[e]
	s.mu.RUnlock()
	return
}

func (s *Service) Update(newConfigs []interface{}) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, nc := range newConfigs {
		if c, ok := nc.(Config); ok {
			s.endpoints[c.Endpoint] = c
		} else {
			return fmt.Errorf("unexpected config object type, got %T exp %T", nc, c)
		}
	}
	return nil
}

func (s *Service) Test(options interface{}) error {
	return nil
}

type testOptions struct{}

func (s *Service) TestOptions() interface{} {
	return &testOptions{}
}

// Prefers URL over Endpoint
func (h *handler) Handle(event alert.Event) {
	var err error

	// Construct the body of the HTTP request
	body := h.bp.Get()
	defer h.bp.Put(body)
	ad := alertDataFromEvent(event)

	err = json.NewEncoder(body).Encode(ad)
	if err != nil {
		h.logger.Printf("E! failed to marshal alert data json: %v", err)
		return
	}

	// Create the HTTP request
	var req *http.Request
	if h.url != "" {
		req, err = http.NewRequest("POST", h.url, body)
		if err != nil {
			h.logger.Printf("E! failed to create POST request: %v", err)
			return
		}
	} else {
		c, ok := h.s.endpoint(h.endpoint)
		if !ok {
			h.logger.Printf("E! endpoint does not exist: %v", h.endpoint)
			return
		}
		req, err = c.NewRequest(body)
	}

	// Execute the request
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		h.logger.Printf("E! failed to POST alert data: %v", err)
		return
	}
	resp.Body.Close()
}

// AlertData is a structure that contains relevant data about an alert event.
// The structure is intended to be JSON encoded, providing a consistent data format.
type AlertData struct {
	ID       string        `json:"id"`
	Message  string        `json:"message"`
	Details  string        `json:"details"`
	Time     time.Time     `json:"time"`
	Duration time.Duration `json:"duration"`
	Level    alert.Level   `json:"level"`
	Data     models.Result `json:"data"`
}

func alertDataFromEvent(event alert.Event) AlertData {
	return AlertData{
		ID:       event.State.ID,
		Message:  event.State.Message,
		Details:  event.State.Details,
		Time:     event.State.Time,
		Duration: event.State.Duration,
		Level:    event.State.Level,
		Data:     event.Data.Result,
	}
}
