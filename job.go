/**
 * @ Author: Michael Harrison
 * @ Create Time: 2024-10-14 09:12:52
 * @ Description: Job.go contains the defintion of the Job and JobResult structs.
 */

package microbatcher

// Job is a struct containing some unknown data for processing, and a channel to return the result.
type Job struct {
	Data     interface{}     // The data to be processed.
	ResultCh chan *JobResult // A channel to return the result of processing the job.
}

// JobResult is a struct containing the result of processing a job along with a status and message (if applicable.)
type JobResult struct {
	Data    interface{} // The data post-processing
	Message string      // A message (if applicable)
	Status  string      // The status of the processing.
}
