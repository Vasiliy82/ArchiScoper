package workflows

import (
	"context"
	"time"

	"github.com/Vasiliy82/ArchiScoper/retailer-api/pkg/domain"
	"github.com/Vasiliy82/ArchiScoper/retailer-api/pkg/tracing"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// OrderWorkflow — главный процесс обработки заказа
func OrderWorkflow(ctx workflow.Context, order domain.Order) error {
	logger := workflow.GetLogger(ctx)
	logger.Info("Запущен OrderWorkflow", "orderID", order.ID)

	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 10 * time.Second,
		RetryPolicy:         &temporal.RetryPolicy{MaximumAttempts: 3},
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	// Последовательные шаги обработки с трассировкой
	err := executeStep(ctx, "AcceptOrder", AcceptOrder, order)
	if err != nil {
		return err
	}

	err = executeStep(ctx, "AssembleOrder", AssembleOrder, order)
	if err != nil {
		return err
	}

	err = executeStep(ctx, "PayOrder", PayOrder, order)
	if err != nil {
		return err
	}

	err = executeStep(ctx, "ShipOrder", ShipOrder, order)
	if err != nil {
		return err
	}

	err = executeStep(ctx, "CompleteOrder", CompleteOrder, order)
	if err != nil {
		return err
	}

	logger.Info("Заказ завершен", "orderID", order.ID)
	return nil
}

// executeStep выполняет шаг обработки заказа с трассировкой
func executeStep(wfctx workflow.Context, stepName string, activityFunc interface{}, order domain.Order) error {
	_, span := tracing.StartApplication(context.Background(), stepName)
	defer span.End()

	span.SetAttributes(tracing.OrderAttributes(order)...)

	err := workflow.ExecuteActivity(wfctx, activityFunc, order).Get(wfctx, &order)
	if err != nil {
		span.RecordError(err)
		return err
	}

	return nil
}
