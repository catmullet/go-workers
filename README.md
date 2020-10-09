![go workers](https://raw.githubusercontent.com/catmullet/go-workers/assets/goworkers.png)

![Code Climate maintainability](https://img.shields.io/codeclimate/maintainability-percentage/catmullet/go-workers)
# Simple Workers Wrapper
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
