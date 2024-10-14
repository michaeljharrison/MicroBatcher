# MicroBatcher

_A simple example library for batch processing of generic data in Go_

## Prerequisites:

- **Go** (_tested on version **1.23.2**_)

## Usage:

### API:

The API is relatively simplistic, you should be able to just submit `Job`s in to the batcher and get the results back as required, they will be automatically processed once one of two thresholds have been reached, either the `MaxBatchSize` has been reached in the current batch or the `MaxBatchLifetime` has expired. Results will be returned as `JobResult`s with the status, message and processed data.

The most important part of the API is to provide a custom `BatchProcessor` as this is the part that actually contains any important business logic to be executed on the batch of jobs.

#### Configuration and Instantiation:

You can call `NewMicroBatcher` to create a new batcher with the only paramter being a `MicroBatcherConfig` object, and call `batcher.Shutdown()` to stop accepting jobs, flush remaining jobs and end the ticker for the batch lifetime.

```go
// Create a new batcher with a custom configuration.
batcher := NewMicroBatcher(MicroBatcherConfig{
	MaxBatchSize:     100,
	MaxBatchLifetime: 2 * time.Second,
	BatchProcessorFn: func(jobs []*Job) {
		// Do something really really important here with the jobs and their data.
	},
})
defer batcher.Shutdown()

```

#### Submitting Jobs:

Jobs are generic objects which contain some `Data` for your provided `BatchProccessor` function to operate on and transform.

Jobs are submitted using a single call and return a `JobResult`:

```go
...
jobRes := batcher.SubmitJob(&Job{Data: {...}})
// The status can be checked to make sure it was processed.
// Note: If the batcher was already shutting down, it will be returned with a rejected status.
if jobRes.Status == JobStatusProcessed {
    // It's processed, you can do something with it here:
    myPostProcessingFunction(jobRes.Data)
}
...
```

Keep in mind, depending on your Batch configuration, the `submitJob` call might block for some time, so you might prefer to run it in a seperate goRoutine and wait for all the jobs in the batch to complete (See examples below)

#### Shutdown

You can shutdown the batcher, which will stop accepting new jobs and flush all already accepted ones. Shutdown will block until the jobs are flushed, but you may try in a seperate goroutine to submit new jobs while it is shutting down. These jobs will be rejected with a status and message.

```go
// Assume here you have already called batcher.Shutdown() in a seperate goroutine.
jobRes := batcher.SubmitJob(&Job{Data: {...}})
fmt.Println(jobRes.Status)  // Will output "Rejected"
fmt.Println(jobRes.Message) // Will output "Batcher is shutting down, no new jobs can be accepted."

```

### Examples:

Below is a simple example to batch some jobs with default settings.

```go
// Create a default batcher, which will flush after 100 jobs or 5 seconds.
batcher := NewMicroBatcher(DefaultBatchConfig())

// Submit a number of jobs async, and wait for them all to be resolved.
wg := sync.WaitGroup{}
for i := 0; i < 5; i++ {
	wg.Add(1)
	go func() {
        // This will not return until the job has been processed in a batch.
		jobRes := batcher.SubmitJob(&Job{})
		wg.Done()
	}()
}
wg.Wait()

// Your jobs have now all been processed.
```

Alternatively, you can provide custom configuration easily:
_NOTE_: Remember, it is important that you provide a processor function to the configuration, otherwise nothing will be achieved!

```go
// Create a default batcher, which will flush after 100 jobs or 2 seconds.
batcher := NewMicroBatcher(MicroBatcherConfig{
	MaxBatchSize:     100,
	MaxBatchLifetime: 2 * time.Second,
	BatchProcessorFn: func(jobs []*Job) {
		// Do something really really important here!
	},
})
defer batcher.Shutdown()

// Submit a number of jobs async, and wait for them all to be resolved.
for i := 0; i < 5; i++ {
	go func() {
		// This will not return until the job has been processed in a batch.
		jobRes := batcher.SubmitJob(&Job{})
		// Do something with the job results here.
	}()
}

// Wait 2 seconds and your jobs will be processed and results returned.
time.Sleep(2 * time.Second)
```

For more examples I suggest looking at the `microbatcher_test` file which contains some simple use cases.

## Testing & Debugging:

All tests will be run automatically on commit in Github Actions.

All unit tests can be found in [`microbatcher_test.go`](./microbatcher_test.go). And cover the core functionality of the API, keep in mind these tests use a dummy `BatchProcessor` and only cover the functionality of the batching and flushing itself.

There are minor logs included using [Zap](https://github.com/uber-go/zap) and are all set to Debug level.

## Dependencies:

The only dependency to this project outside of core libraries is [Zap](https://github.com/uber-go/zap) to provide some nicer and more cloud friendly logs, however these are not essential to the library and could be easily replaced with a preferred logging library (or none) as desired.
