![go workers](https://raw.githubusercontent.com/catmullet/go-workers/assets/constworker_header_anim.gif)

[![Maintainability](https://api.codeclimate.com/v1/badges/402fee86fbd1e24defb2/maintainability)](https://codeclimate.com/github/catmullet/go-workers/maintainability) [![Go Report Card](https://goreportcard.com/badge/github.com/catmullet/go-workers)](https://goreportcard.com/report/github.com/catmullet/go-workers)    

# Examples
* [Quickstart](https://github.com/catmullet/go-workers/blob/master/examples/quickstart/quickstart.go)
* [Multiple Go Workers](https://github.com/catmullet/go-workers/blob/master/examples/multiple_workers/multipleworkers.go)
* [Passing Fields](https://github.com/catmullet/go-workers/blob/master/examples/passing_fields/passingfields.go)
# Getting Started
### Pull in the dependency
```zsh
go get github.com/catmullet/go-workers
```

### Add the import to your project
giving an alias helps since go-workers doesn't exactly follow conventions.    
_(If your using a JetBrains IDE it should automatically give it an alias)_
```go
import (
    worker "github.com/catmullet/go-workers"
)
```
### Create a new worker <img src="https://raw.githubusercontent.com/catmullet/go-workers/assets/constworker.png" alt="worker" width="35"/>
The NewWorker factory method returns a new worker.    
_(Method chaining can be performed on this method like calling .Work() immediately after.)_
```go
worker := worker.NewWorker(ctx, workerFunction, numberOfWorkers)
```
### Send work to worker
Send accepts an interface.  So send it anything you want.
```go
worker.Send("Hello World")
```
### Close the worker when done
This closes the in channel on the worker and signals to the go functions to stop.
```go
worker.Close()
```
### Wait for the worker to finish and handle errors
Any error that bubbles up from your worker functions will return here.
```go
if err := worker.Wait(); err != nil {
    //Handle error
}
```

## Working With Multiple Workers
### Passing work form one worker to the next 

By using the InFrom method you can tell `workerTwo` to accept output from `workerOne`
```go
workerOne := worker.NewWorker(ctx, workerOneFunction, 100).Work()
workerTwo := worker.NewWorker(ctx, workerTwoFunction, 100).InFrom(workerOne).Work()
```
### Accepting output from multiple workers
It is possible to accept output from more than one worker but it is up to you to determine what is coming from which worker.
```go
workerOne := worker.NewWorker(ctx, workerOneFunction, 100).Work()
workerTwo := worker.NewWorker(ctx, workerOneFunction, 100).Work()
workerThree := worker.NewWorker(ctx, workerTwoFunction, 100).InFrom(workerOne, workerTwo).Work()
```

## Passing Fields To Workers
### Adding Field Values
Fields can be passed to worker functions as configuration values via the `AddField` method.  It accepts a key and a value.
If you are passing a struct it should likely be a Pointer.
Fields although concurrent safe should only be used for configuration at the start of your worker function.

<img src="https://raw.githubusercontent.com/catmullet/go-workers/assets/constworker2.png" alt="worker" width="35"/> **ONLY** use the `Send()` method to get data into your worker. It is not shared memory unlike the fields.
```go
worker := worker.NewWorker(ctx, workerFunction, 100).AddField("message", "Hello World")
```

### Getting Field Values
To get the fields in the worker function use the `BindField` method.
Only use this function outside of the for loop. Create a variable of the same type you are trying to get out of fields and pass a pointer of it into the `BindField` method along with the key.

```go
func workerFunction(w *worker.Worker) error {

    // Pull in fields outside of for loop only.
    var message string
    w.BindField("message", &message)

    for in := range w.In() {
        // Use message variable here
    }
    return nil
}
```

### Setting Timeouts or Deadlines
If your workers needs to stop at a deadline or you just need to have a timeout use the SetTimeout or SetDeadline methods.
Worker functions backbone is errgroups, because of this when you look for the signal (IsDone() as seen below), you need to return an error
and handle it otherwise the errgroup will wait for your worker functions to be finish causing deadlock.
```go
 // Setting a timeout of 2 seconds
 timeoutWorker.SetTimeout(2 * time.Second)

 // Setting a deadline of 4 hours from now
 deadlineWorker.SetDeadline(time.Now().Add(4 * time.Hour))

func workerFunction(w *worker.Worker) error {
	for {
		select {
		case in := <-w.In():
			fmt.Println(in)
			time.Sleep(1 * time.Second)
		case <-w.IsDone():
			// due to the nature of errgroups in order to stop the worker from
			// waiting an error needs to be returned
			return fmt.Errorf("timeout reached")
		}
	}
}
```
