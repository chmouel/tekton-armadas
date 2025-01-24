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
	"github.com/openshift-pipelines/tekton-armadas/pkg/types"
	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

func (c *controller) doTypes(ctx context.Context, aEvent types.ArmadaEvent) error {
	tt, err := types.ReadTektonTypes(ctx, []string{aEvent.PipelineRun})
	if err != nil {
		return fmt.Errorf("failed to read tekton types: %w", err)
	}

	for _, pr := range tt.Tekton.PipelineRuns {
		if _, err := c.clients.Tekton.TektonV1().PipelineRuns(aEvent.Namespace).Get(ctx, pr.GetName(), metav1.GetOptions{}); err == nil {
			if err := c.clients.Tekton.TektonV1().PipelineRuns(aEvent.Namespace).Delete(ctx, pr.GetName(), metav1.DeleteOptions{}); err != nil {
				return fmt.Errorf("error deleting pipelinerun %s: %w", pr.GetName(), err)
			} else {
				c.logger.Info(fmt.Sprintf("pipelinerun %s has been delete", pr.GetName()))
			}
		}

		if cp, err := c.clients.Tekton.TektonV1().PipelineRuns(aEvent.Namespace).Create(ctx, pr, metav1.CreateOptions{}); err != nil {
			return fmt.Errorf("error creating pipelinerun: %w", err)
		} else {
			c.logger.Info(fmt.Sprintf("pipelinerun %s has been created", cp.GetName()))
		}
	}
	return err
}

func (c *controller) handleEvent(ctx context.Context) http.HandlerFunc {
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

		aEvent := types.ArmadaEvent{}
		if err := event.DataAs(&aEvent); err != nil {
			c.logger.Errorf("failed to convert event data: %v", err)
		}

		if err := c.doTypes(ctx, aEvent); err != nil {
			c.logger.Errorf("failed to do types: %+v", err)
			c.writeResponse(response, http.StatusInternalServerError, "failed to read tekton types")
			return
		}

		// output a json message with details
		c.writeResponse(response, http.StatusAccepted, fmt.Sprintf(`{"message": "created", "event": %s}`, event.Context.GetID()))
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
