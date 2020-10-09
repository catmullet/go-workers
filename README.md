![go workers](https://raw.githubusercontent.com/catmullet/go-workers/assets/goworkers.png)

# Simple Workers Wrapper [![Maintainability](https://api.codeclimate.com/v1/badges/402fee86fbd1e24defb2/maintainability)](https://codeclimate.com/github/catmullet/go-workers/maintainability) [![Go Report Card](https://goreportcard.com/badge/github.com/catmullet/go-workers)](https://goreportcard.com/report/github.com/catmullet/go-workers)
Wrapping concurrent functions in a goworker wrapper makes it clean, safe and easy.
```go
    worker := goworker.NewWorker(ctx, workerFunction, 10).Work()

    func workerFunction(w *goworker.Worker) error {
    	for in := range w.In() {
        // Do work here with input
            fmt.Prinln(in)
            
        // Out to another worker
            w.Out(<output>)
    	}
    	return nil
    }
```
# Usage
* Safely Running groups of workers concurrently or consecutively that require input and output from channels
# Examples
* ![Quickstart](https://github.com/catmullet/go-workers/blob/master/examples/quickstart/quickstart.go)
* ![Multiple Go Workers](https://github.com/catmullet/go-workers/blob/master/examples/multiple_workers/multipleworkers.go)
* ![Passing Fields](https://github.com/catmullet/go-workers/blob/master/examples/passing_fields/passingfields.go)
