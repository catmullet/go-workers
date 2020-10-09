![go workers](https://raw.githubusercontent.com/catmullet/go-workers/assets/goworkers.png)
# Simple Workers Wrapper
Wrapping concurrent functions in a goworker wrapper makes it clean, safe and easy.
```go
    worker := goworker.NewWorker(ctx, workerFunction, 10).Work()

    func workerFunction(w *goworker.Worker) error {
    	for in := range w.In() {
    		// Do work here
    	}
    	return nil
    }
```
# Usage
* Safely Running groups of workers concurrently or consecutively that require input and output from channels
# Examples

