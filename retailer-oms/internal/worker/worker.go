package worker

import (
	"log"

	"github.com/Vasiliy82/ArchiScoper/retailer-oms/internal/workflows"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
)

// StartWorker запускает Temporal Worker
func StartWorker() {
	c, err := client.Dial(client.Options{})
	if err != nil {
		log.Fatalf("Error connecting to Temporal: %v", err)
	}
	defer c.Close()

	w := worker.New(c, "order-tasks", worker.Options{})

	w.RegisterWorkflow(workflows.OrderWorkflow)
	w.RegisterActivity(workflows.AcceptOrder)
	w.RegisterActivity(workflows.AssembleOrder)
	w.RegisterActivity(workflows.PayOrder)
	w.RegisterActivity(workflows.ShipOrder)
	w.RegisterActivity(workflows.CompleteOrder)

	log.Println("Worker started...")
	if err := w.Run(worker.InterruptCh()); err != nil {
		log.Fatalf("Error running worker: %v", err)
	}
}
