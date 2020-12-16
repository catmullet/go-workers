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
type MyWorker struct {}

func NewMyWorker() *MyWorker {
	return &MyWorker{}
}

func (my *MyWorker) Work(w *goworker.Worker, in interface{}) error {
	// work iteration here
}

worker := worker.NewWorker(ctx, NewMyWorker(), numberOfWorkers)
```
### Send work to worker
Send accepts an interface.  So send it anything you want.
```go
worker.Send("Hello World")
```
### Close and wait for the worker to finish and handle errors
Any error that bubbles up from your worker functions will return here.
```go
if err := worker.Close(); err != nil {
    //Handle error
}
```

## Working With Multiple Workers
### Passing work form one worker to the next 

By using the InFrom method you can tell `workerTwo` to accept output from `workerOne`
```go
workerOne := worker.NewWorker(ctx, NewMyWorker(), 100).Work()
workerTwo := worker.NewWorker(ctx, NewMyWorkerTwo(), 100).InFrom(workerOne).Work()
```
### Accepting output from multiple workers
It is possible to accept output from more than one worker but it is up to you to determine what is coming from which worker.
```go
workerOne := worker.NewWorker(ctx, NewMyWorker(), 100).Work()
workerTwo := worker.NewWorker(ctx, NewMyWorkerTwo(), 100).Work()
workerThree := worker.NewWorker(ctx, NewMyWorkerThree(), 100).InFrom(workerOne, workerTwo).Work()
```

## Passing Fields To Workers
### Adding Values
Fields can be passed via the workers object.

<img src="https://raw.githubusercontent.com/catmullet/go-workers/assets/constworker2.png" alt="worker" width="35"/> **ONLY** use the `Send()` method to get data into your worker. It is not shared memory unlike the worker objects values.
```go
type MyWorker struct {
	message string
}

func NewMyWorker(message string) *MyWorker {
	return &MyWorker{message}
}

func (my *MyWorker) Work(w *goworker.Worker, in interface{}) error {
	fmt.Println(my.message)
}

worker := worker.NewWorker(ctx, NewMyWorker(), 100).Work()
```

### Setting Timeouts or Deadlines
If your workers needs to stop at a deadline or you just need to have a timeout use the SetTimeout or SetDeadline methods.
```go
 // Setting a timeout of 2 seconds
 timeoutWorker.SetTimeout(2 * time.Second)

 // Setting a deadline of 4 hours from now
 deadlineWorker.SetDeadline(time.Now().Add(4 * time.Hour))

func workerFunction(w *worker.Worker, in interface{}) error {
	fmt.Println(in)
	time.Sleep(1 * time.Second)
}
```
