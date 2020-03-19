package common

import (
	"fmt"
	"sync"
	"time"
)

type Job struct {
	Type string
	Content string
	Code string
}

type Runner func(job Job) ([]byte, error)

type Verifier func(actual string) bool

type JobResult struct {
	Job    Job
	Output string
	Result bool
	ExecutionTime time.Duration
}

func Consume(verifier Verifier, runners []Runner, jobs []Job) chan JobResult {
	fmt.Printf("------------------------------ SUMMARY ------------------------------\n")
	fmt.Printf("number of jobs to be consumed: %d\n", len(jobs))
	fmt.Printf("number of runners: %d\n", len(runners))
	fmt.Printf("---------------------------------------------------------------------\n\n")
	jobChan := make(chan Job)
	resultChan := make(chan JobResult, 1000)

	var wg sync.WaitGroup
	// start all workers
	for k, v := range runners {
		go func(index int, runner Runner) {
			for {
				fmt.Printf("RUNNER %d waiting for more Job\n", index)
				job := <- jobChan
				fmt.Printf("RUNNER %d received Job %s\n", index, job.Code)
				start := time.Now()
				output, err := runner(job)
				fmt.Printf("RUNNER %d finished Job %s\n", index, job.Code)
				if err != nil {
					resultChan <- JobResult{
						Job:    job,
						Output: string(output),
						Result: false,
						ExecutionTime: time.Since(start),
					}
					wg.Done()
					continue
				}
				result := verifier(string(output))
				resultChan <- JobResult{
					Job:    job,
					Output: string(output),
					Result: result,
					ExecutionTime: time.Since(start),
				}
				wg.Done()
			}
		}(k, v)
	}

	go func() {
		wg.Add(len(jobs))
		// fetch jobs to workers
		for k, v := range jobs {
			fmt.Printf("CONSUMER waiting for fetching %d of %d to workers, Job code = %s\n", k, len(jobs), v.Code)
			jobChan <- v
			fmt.Printf("CONSUMER fetched %d of %d to workers, Job code = %s\n", k, len(jobs), v.Code)
		}
		wg.Wait()
		close(resultChan)
	}()

	return resultChan
}