package queue

import (
	"context"
	"fmt"
	"oblak/internal/function"
	"oblak/internal/vm/config"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-amqp/v3/pkg/amqp"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
)

type VmRouter struct {
	router       *message.Router
	dlqPublisher message.Publisher
}

func NewVmRouter(cfg config.Config, build *BuildHandler, execute *ExecuteHandler, store *function.Store) (*VmRouter, error) {
	logger := watermill.NewStdLogger(true, false)

	baseDLQPublisher, err := amqp.NewPublisher(PublisherConfig(cfg), logger)
	if err != nil {
		return nil, fmt.Errorf("dlq publisher: %w", err)
	}
	dlqPublisher := &statusFailPublisher{base: baseDLQPublisher, store: store}

	poisonMiddleware, err := middleware.PoisonQueue(dlqPublisher, cfg.BuildDLQName)
	if err != nil {
		_ = dlqPublisher.Close()
		return nil, fmt.Errorf("poison queue middleware: %w", err)
	}

	router, err := message.NewRouter(message.RouterConfig{}, logger)
	if err != nil {
		_ = dlqPublisher.Close()
		return nil, fmt.Errorf("router: %w", err)
	}

	router.AddMiddleware(
		middleware.Retry{
			MaxRetries:      3,
			InitialInterval: 500 * time.Millisecond,
			MaxInterval:     30 * time.Second,
			Multiplier:      2,
			Logger:          logger,
		}.Middleware,
		middleware.Recoverer,
	)

	for i := range cfg.MaxConcurrentBuilds {
		sub, err := amqp.NewSubscriber(PrepareEnvConsumerConfig(cfg), logger)
		if err != nil {
			_ = dlqPublisher.Close()
			_ = router.Close()
			return nil, fmt.Errorf("build subscriber: %w", err)
		}
		h := router.AddConsumerHandler(fmt.Sprintf("build-%d", i), cfg.BuildQueueName, sub, build.Handle)
		h.AddMiddleware(poisonMiddleware)
	}

	for i := range cfg.MaxConcurrentExecutes {
		sub, err := amqp.NewSubscriber(ExecuteConsumerConfig(cfg), logger)
		if err != nil {
			_ = dlqPublisher.Close()
			_ = router.Close()
			return nil, fmt.Errorf("execute subscriber: %w", err)
		}
		router.AddConsumerHandler(fmt.Sprintf("execute-%d", i), cfg.ExecuteQueueName, sub, execute.Handle)
	}

	return &VmRouter{router: router, dlqPublisher: dlqPublisher}, nil
}

func (r *VmRouter) Run(ctx context.Context) error {
	return r.router.Run(ctx)
}

func (r *VmRouter) Close() error {
	_ = r.dlqPublisher.Close()
	return r.router.Close()
}
