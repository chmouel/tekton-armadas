package minion

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/openshift-pipelines/tekton-armadas/pkg/clients"
	"github.com/openshift-pipelines/tekton-armadas/pkg/templates"
	"go.uber.org/zap"
	"knative.dev/eventing/pkg/adapter/v2"
	"knative.dev/pkg/logging"
	"knative.dev/pkg/system"
)

const (
	globalControllerPort = "8081"
	httpTimeoutHandler   = 600 * time.Second
)

// controller generates events at a regular interval.
type controller struct {
	logger   *zap.SugaredLogger
	clients  *clients.Clients
	interval time.Duration
}

type envConfig struct {
	adapter.EnvConfig
}

func NewEnvConfig() adapter.EnvConfigAccessor {
	return &envConfig{
		adapter.EnvConfig{
			Namespace: system.Namespace(),
		},
	}
}

type Response struct {
	Status  int    `json:"status"`
	Message string `json:"message"`
}

func (c *controller) writeResponse(response http.ResponseWriter, statusCode int, message string) {
	response.WriteHeader(statusCode)
	response.Header().Set("Content-Type", "application/json")
	body := Response{
		Status:  statusCode,
		Message: message,
	}
	if err := json.NewEncoder(response).Encode(body); err != nil {
		c.logger.Errorf("failed to write back sink response: %v", err)
	}
}

func (c *controller) handleEvent(_ context.Context) http.HandlerFunc {
	return func(response http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost {
			c.writeResponse(response, http.StatusOK, "ok")
			return
		}

		event, err := cloudevents.NewEventFromHTTPRequest(request)
		if err != nil {
			c.logger.Errorf("failed to create event from request: %v", err)
			response.WriteHeader(http.StatusInternalServerError)
		}
		c.logger.Debugf("Received event: %s", event.String())

		aEvent := templates.ArmadaEvent{}
		if err := event.DataAs(&aEvent); err != nil {
			c.logger.Errorf("failed to convert event data: %v", err)
		}

		c.writeResponse(response, http.StatusOK, "skipped event")
	}
}

func (c *controller) Start(ctx context.Context) error {
	controllerPort := globalControllerPort
	envControllerPort := os.Getenv("ARMADA_MINION_CONTROLLER_PORT")
	if envControllerPort != "" {
		controllerPort = envControllerPort
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/live", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, "ok")
	})

	mux.HandleFunc("/", c.handleEvent(ctx))

	//nolint: gosec
	srv := &http.Server{
		Addr:    ":" + controllerPort,
		Handler: http.TimeoutHandler(mux, httpTimeoutHandler, "Listener Timeout!\n"),
	}

	return srv.ListenAndServe()
}

func NewController(clients *clients.Clients) adapter.AdapterConstructor {
	return func(ctx context.Context, _ adapter.EnvConfigAccessor, _ cloudevents.Client) adapter.Adapter {
		return &controller{
			logger:   logging.FromContext(ctx),
			clients:  clients,
			interval: 5 * time.Second,
		}
	}
}
