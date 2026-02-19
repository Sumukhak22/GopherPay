package worker

import (
	"context"
	"log/slog"
	"sync"

	"gopherpay/internal/billing"
)

type TransferJob struct {
	Request billing.TransferRequest
}

type Pool struct {
	jobs    chan TransferJob
	service *billing.Service
	logger  *slog.Logger
	wg      sync.WaitGroup
}

func NewPool(bufferSize int, service *billing.Service, logger *slog.Logger) *Pool {
	return &Pool{
		jobs:    make(chan TransferJob, bufferSize),
		service: service,
		logger:  logger,
	}
}

func (p *Pool) Start(workerCount int) {
	for i := 0; i < workerCount; i++ {
		p.wg.Add(1)
		go p.worker()
	}
}

func (p *Pool) worker() {
	defer p.wg.Done()

	for job := range p.jobs {
		err := p.service.Transfer(context.Background(), job.Request)
		if err != nil {
			p.logger.Error("transfer processing failed",
				"request_id", job.Request.RequestID,
				"error", err,
			)
		}
	}
}

func (p *Pool) Submit(job TransferJob) bool {
	select {
	case p.jobs <- job:
		return true
	default:
		return false
	}
}

func (p *Pool) Shutdown() {
	close(p.jobs)
	p.wg.Wait()
}
