/**
 * @ Author: Michael Harrison
 * @ Create Time: 2024-10-14 09:14:23
 * @ Description: microbatcher.go contains the main functionality for the microbatcher package.
 */

// The microbatcher package provides functionality to package jobs as batches to improve throughput.
package microbatcher

import (
	"sync"
	"time"

	"go.uber.org/zap"
)

const (
	DefaultMaxBatchSize     = 100
	DefaultMaxBatchLifetime = 5 * time.Second
	JobStatusProcessed      = "Processed"
	JobStatusRejected       = "Rejected"
	JobMessageShuttingDown  = "Batcher is shutting down, no new jobs can be accepted."
)

// MicroBatcherConfig is a struct containing the configuration for the batcher.
type MicroBatcherConfig struct {
	MaxBatchSize     int           // The maximum number of jobs in a batch before flushing.
	MaxBatchLifetime time.Duration // The maximum lifetime of the batch before flushing.
	BatchProcessorFn func([]*Job)  // The downstream function which will process a given batch.
}

// DefaultBatchConfig returns a default, generic configuration for the processor
func DefaultBatchConfig() MicroBatcherConfig {
	return MicroBatcherConfig{
		MaxBatchSize:     DefaultMaxBatchSize,
		MaxBatchLifetime: DefaultMaxBatchLifetime,
		BatchProcessorFn: DefaultBatchProcessor,
	}
}

// DefaultBatchProcessor simply marks the jobs as processed, this should be replaced with a custom processor in any real use case.
func DefaultBatchProcessor(jobs []*Job) {
	for _, job := range jobs {
		job.Data = JobStatusProcessed
	}
}

// MicroBatcher is a struct containing the current state of the microbatcher and it's config.
type MicroBatcher struct {
	config       MicroBatcherConfig // The config for the batcher.
	currentBatch []*Job             // The batch to be processed.
	ticker       *time.Ticker       // The ticker for the batch to be flushed after
	shuttingDown bool               // A flag for the shutdown procedure to stop accepting new jobs.
	mu           sync.Mutex         // A mutex to protect the state of the current batch while being flushed.
}

// NewMicroBatcher creates and returns a new microbatcher with the provided configuration.
func NewMicroBatcher(config MicroBatcherConfig) *MicroBatcher {
	zap.ReplaceGlobals(zap.Must(zap.NewProduction()))
	mb := &MicroBatcher{
		config:       config,
		currentBatch: []*Job{},
		shuttingDown: false,
	}

	// Add a loop with a ticker to flush the batch after a given lifetime.
	go func() {
		mb.ticker = time.NewTicker(mb.config.MaxBatchLifetime)
		defer mb.ticker.Stop()
		for {
			select {
			case <-mb.ticker.C:
				zap.L().Debug("Batch lifetime expired, flushing...")
				if len(mb.currentBatch) > 0 {
					mb.FlushBatch()
					mb.ticker.Reset(mb.config.MaxBatchLifetime)
				}
			}
		}
	}()
	return mb
}

// SubmitJob adds a new job to the current batch, flushes if required.
func (mb *MicroBatcher) SubmitJob(job *Job) *JobResult {
	// Do not accept any new jobs while in the process of shutting down.
	if mb.shuttingDown {
		return &JobResult{Status: JobStatusRejected, Message: JobMessageShuttingDown}
	}

	mb.mu.Lock()
	job.ResultCh = make(chan *JobResult)
	mb.currentBatch = append(mb.currentBatch, job)
	zap.L().Debug("Job added to batch", zap.Any("Batch length:", len(mb.currentBatch)))
	mb.mu.Unlock()
	if len(mb.currentBatch) >= mb.config.MaxBatchSize {
		go mb.FlushBatch()
	}

	result := <-job.ResultCh
	close(job.ResultCh)
	return result
}

// FlushBatch processes the current batch, creates the job result and then returns via the jobs channels.
func (mb *MicroBatcher) FlushBatch() {
	mb.mu.Lock()
	// Check if there is anything to flush first.
	if len(mb.currentBatch) == 0 {
		mb.mu.Unlock()
		return
	}

	zap.L().Debug("Flushing batch...", zap.Any("Batch length:", len(mb.currentBatch)))
	// Batch processor should process the current batch, and place any resulting information within the job data field.
	mb.config.BatchProcessorFn(mb.currentBatch)

	// Iterate through the jobs and update their result.
	for _, job := range mb.currentBatch {
		result := &JobResult{
			Data:   job.Data,
			Status: JobStatusProcessed,
		}
		job.ResultCh <- result
	}

	// Clear the current batch and releaes lock.
	mb.currentBatch = []*Job{}
	mb.mu.Unlock()
}

// Shutdown stops the MicroBatcher from accepting new jobs, and flushes any existing jobs.
func (mb *MicroBatcher) Shutdown() {
	zap.L().Debug("Shutting down batcher...")
	mb.shuttingDown = true
	if mb.ticker != nil {
		mb.ticker.Stop()
	}
	mb.FlushBatch()
}
