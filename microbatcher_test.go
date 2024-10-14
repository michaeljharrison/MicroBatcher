/**
 * @ Author: Michael Harrison
 * @ Create Time: 2024-10-14 09:13:11
 * @ Description: microbatcher_test.go contains our unit tests for the microbatcher package.
 */

package microbatcher

import (
	"sync"
	"testing"
	"time"
)

// TestNewMicroBatcher creates a new microbatcher and checks the config.
func TestNewMicroBatcher(t *testing.T) {
	config := DefaultBatchConfig()
	batcher := NewMicroBatcher(DefaultBatchConfig())

	if batcher.config.MaxBatchSize != config.MaxBatchSize {
		t.Errorf("Expected MaxBatchSize to be %d, got %d", config.MaxBatchSize, batcher.config.MaxBatchSize)
	}
	if batcher.config.MaxBatchLifetime != config.MaxBatchLifetime {
		t.Errorf("Expected MaxBatchLifetime to be %v, got %v", config.MaxBatchLifetime, batcher.config.MaxBatchLifetime)
	}
}

// TestMicroBatcher_SubmitSingleJob submits a single job and checks it is processed.
func TestMicroBatcher_SubmitSingleJob(t *testing.T) {

	// Create a single batch size as we just want to see the flush in this test.
	batcher := NewMicroBatcher(MicroBatcherConfig{
		MaxBatchSize:     1,
		MaxBatchLifetime: 1 * time.Minute,
		BatchProcessorFn: func(jobs []*Job) {
			// Do absolutely nothing here.
		},
	})
	defer batcher.Shutdown()

	//1. Submit a single job and check it has been processed.
	jobRes := batcher.SubmitJob(&Job{Data: "test1"})

	if jobRes.Status != JobStatusProcessed {
		t.Errorf("Expected job to be %s, got %s", JobStatusProcessed, jobRes.Status)
	}
}

// TestMicroBatcher_SubmitMultipleJobs submits multiple jobs and checks they are processed when flushed via Batch size limit.
func TestMicroBatcher_TriggerBatchSize(t *testing.T) {

	// Create a batch size of 5 with a high lifetime to see it flush via size.
	batcher := NewMicroBatcher(MicroBatcherConfig{
		MaxBatchSize:     5,
		MaxBatchLifetime: 5 * time.Minute,
		BatchProcessorFn: func(jobs []*Job) {
			// Do absolutely nothing here.
		},
	})
	defer batcher.Shutdown()

	//1. Submit 5 jobs at once, check the batch size after each.
	wg := sync.WaitGroup{}
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			jobRes := batcher.SubmitJob(&Job{})
			if jobRes.Status != JobStatusProcessed {
				t.Errorf("Expected job to be %s, got %s", JobStatusProcessed, jobRes.Status)
			}
			wg.Done()
		}()
	}
	wg.Wait()

	//2. Batch should now be flushed and empty.
	if len(batcher.currentBatch) != 0 {
		t.Errorf("Expected current batch to be length 0, got %d", len(batcher.currentBatch))
	}
}

// TestMicroBatcher_TriggerBatchLifetime submits multiple jobs and checks they are processed when flushed via Batch lifetime.
func TestMicroBatcher_TriggerBatchLifetime(t *testing.T) {

	// Create a high batch size with a lifetime of 3 seconds to see it flush via lifetime.
	batcher := NewMicroBatcher(MicroBatcherConfig{
		MaxBatchSize:     100,
		MaxBatchLifetime: 3 * time.Second,
		BatchProcessorFn: func(jobs []*Job) {
			// Do absolutely nothing here.
		},
	})
	defer batcher.Shutdown()

	//1. Submit 5 jobs at once,
	for i := 0; i < 5; i++ {
		go func() {

			jobRes := batcher.SubmitJob(&Job{})
			if jobRes.Status != JobStatusProcessed {
				t.Errorf("Expected job to be %s, got %s", JobStatusProcessed, jobRes.Status)
			}
		}()
	}

	//2.  Wait a second for jobs to in the current batch:
	time.Sleep(1 * time.Second)
	if len(batcher.currentBatch) != 5 {
		t.Errorf("Expected current batch to be length 5, got %d", len(batcher.currentBatch))
	}

	//3. Wait three more seconds for batch to flush.
	time.Sleep(3 * time.Second)

	//4. Batch should now be flushed and empty.
	if len(batcher.currentBatch) != 0 {
		t.Errorf("Expected current batch to be length 0, got %d", len(batcher.currentBatch))
	}
}

// TestMicroBatcher_ManuallyFlushBatch submits multiple jobs and checks they are processed when flushed manually.
func TestMicroBatcher_ManuallyFlushBatch(t *testing.T) {
	// Create a high batch size with a long lifetime to see it flush manually
	batcher := NewMicroBatcher(MicroBatcherConfig{
		MaxBatchSize:     100,
		MaxBatchLifetime: 30 * time.Second,
		BatchProcessorFn: func(jobs []*Job) {
			// Do absolutely nothing here.
		},
	})
	defer batcher.Shutdown()

	//1. Submit 5 jobs at once,
	for i := 0; i < 5; i++ {
		go func() {

			jobRes := batcher.SubmitJob(&Job{})
			if jobRes.Status != JobStatusProcessed {
				t.Errorf("Expected job to be %s, got %s", JobStatusProcessed, jobRes.Status)
			}
		}()
	}

	//2.  Wait a second for jobs to in the current batch:
	time.Sleep(1 * time.Second)
	if len(batcher.currentBatch) != 5 {
		t.Errorf("Expected current batch to be length 5, got %d", len(batcher.currentBatch))
	}

	//3. Manually flush the batch.
	batcher.FlushBatch()
	if len(batcher.currentBatch) != 0 {
		t.Errorf("Expected current batch to be length 0, got %d", len(batcher.currentBatch))
	}
}

// TestMicroBatcher_ShutdownFlushesBatch submits multiple jobs, shuts down and checks they are processed when flushed via shutdown.
func TestMicroBatcher_ShutdownFlushesBatch(t *testing.T) {
	// Create a high batch size with a long lifetime to see it flush via shutdown.
	batcher := NewMicroBatcher(MicroBatcherConfig{
		MaxBatchSize:     100,
		MaxBatchLifetime: 30 * time.Second,
		BatchProcessorFn: func(jobs []*Job) {
			// Do absolutely nothing here.
		},
	})
	defer batcher.Shutdown()

	//1. Submit 5 jobs at once,
	for i := 0; i < 5; i++ {
		go func() {

			jobRes := batcher.SubmitJob(&Job{})
			if jobRes.Status != JobStatusProcessed {
				t.Errorf("Expected job to be %s, got %s", JobStatusProcessed, jobRes.Status)
			}
		}()
	}

	//2.  Wait a second for jobs to in the current batch:
	time.Sleep(1 * time.Second)
	if len(batcher.currentBatch) != 5 {
		t.Errorf("Expected current batch to be length 5, got %d", len(batcher.currentBatch))
	}

	//3. Start shutdown to flush batch.
	batcher.Shutdown()

	//4. Batch should now be flushed and empty.
	if len(batcher.currentBatch) != 0 {
		t.Errorf("Expected current batch to be length 0, got %d", len(batcher.currentBatch))
	}
}

// TestMicroBatcher_ShutdownBlocksNewJobs submits multiple jobs, shuts down and checks that a subsequent job is rejected.
func TestMicroBatcher_ShutdownBlocksNewJobs(t *testing.T) {
	// Create a high batch size with a long lifetime so we can see rejected jobs.
	batcher := NewMicroBatcher(MicroBatcherConfig{
		MaxBatchSize:     100,
		MaxBatchLifetime: 30 * time.Second,
		BatchProcessorFn: func(jobs []*Job) {
			// Do absolutely nothing here.
		},
	})
	defer batcher.Shutdown()

	//1. Submit 5 jobs at once,
	for i := 0; i < 5; i++ {
		go func() {

			jobRes := batcher.SubmitJob(&Job{})
			if jobRes.Status != JobStatusProcessed {
				t.Errorf("Expected job to be %s, got %s", JobStatusProcessed, jobRes.Status)
			}
		}()
	}

	//2.  Wait a second for jobs to in the current batch:
	time.Sleep(1 * time.Second)
	if len(batcher.currentBatch) != 5 {
		t.Errorf("Expected current batch to be length 5, got %d", len(batcher.currentBatch))
	}

	//3. Start shutdown in one thread.
	batcher.Shutdown()

	//4. Submit a job in another thread, the result should come back rejected.
	jobRes := batcher.SubmitJob(&Job{})
	if jobRes.Status != JobStatusRejected {
		t.Errorf("Expected job to be %s, got %s", JobStatusRejected, jobRes.Status)
	}
}
