package main

import (
	"context"
	"log"
	"net/http"
	"strings"
	"time"

	"example.com/port-yard-condition-service/config"
	"example.com/port-yard-condition-service/httpapi"
	"example.com/port-yard-condition-service/notify"
	"example.com/port-yard-condition-service/readings"
	"example.com/port-yard-condition-service/schedule"
	"example.com/port-yard-condition-service/store"
	"example.com/port-yard-condition-service/web"
)

func main() {
	address := ":" + config.Port()

	yardHandler := httpapi.NewHandler(store.New(), web.FS)

	opsService := newOpsService(seedOpsRecords())
	opsMux := newOpsMux(opsService)

	readingsService := readings.NewService(readings.NewStore(200))
	readingsMux := readings.NewMux(readingsService)

	logSender := notify.NewLogSender(1000)
	dispatcher := notify.NewDispatcher([]notify.Sender{notify.NewRetrySender(logSender, 3)}, 128, 2)
	dispatcher.Start()
	defer dispatcher.Stop()

	runner := schedule.NewRunner(
		schedule.Job{Name: "ops-audit-sweep", Interval: 15 * time.Minute, Run: func(ctx context.Context) error {
			return trimOpsAudit(ctx, opsService, 500)
		}},
		schedule.Job{Name: "readings-sweep", Interval: 15 * time.Minute, Run: func(ctx context.Context) error {
			return trimReadings(ctx, readingsService)
		}},
	)
	runner.Start()
	defer runner.Stop()

	handler := newRootHandler(yardHandler, opsMux, readingsMux)
	log.Printf("port-yard-condition-service listening on %s", address)
	if err := serveAddress(address, handler); err != nil {
		log.Fatal(err)
	}
}

// newRootHandler routes requests between the yard-zone, ops and readings APIs.
// Each request keeps its own context so that a client cancellation only affects
// the request that was cancelled — it never leaks into subsequent requests.
func newRootHandler(yard http.Handler, ops http.Handler, readings http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/api/ops"):
			ops.ServeHTTP(w, r)
		case strings.HasPrefix(r.URL.Path, "/api/readings"):
			readings.ServeHTTP(w, r)
		default:
			yard.ServeHTTP(w, r)
		}
	})
}

// trimOpsAudit bounds the in-memory audit trail to keep long-running memory
// usage predictable. It honors the job context so a cancelled sweep stops
// instead of mutating the audit trail after the runner has moved on.
func trimOpsAudit(ctx context.Context, service *OpsService, max int) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	service.audit.Trim(max)
	return nil
}

// trimReadings bounds the retained readings per zone.
func trimReadings(ctx context.Context, service *readings.Service) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	return service.Trim()
}
