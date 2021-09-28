# gorkers

<!-- [![Mentioned in Awesome Go](https://awesome.re/mentioned-badge-flat.svg)](https://github.com/avelino/awesome-go#goroutines) -->

<!-- [![Maintainability](https://api.codeclimate.com/v1/badges/402fee86fbd1e24defb2/maintainability)](https://codeclimate.com/github/catmullet/go-workers/maintainability) -->

[![CodeQL](https://github.com/guilhem/gorkers/workflows/CodeQL/badge.svg)](https://github.com/guilhem/gorkers/actions?query=workflow%3ACodeQL)
[![GoCover](http://gocover.io/_badge/github.com/guilhem/gorkers)](http://gocover.io/github.com/guilhem/gorkers)
[![Go Reference](https://pkg.go.dev/badge/github.com/guilhem/gorkers.svg)](https://pkg.go.dev/github.com/guilhem/gorkers)

## Examples

- [Quickstart](https://github.com/guilhem/gorkers/blob/master/examples/quickstart/quickstart.go)
- [Multiple Go Workers](https://github.com/guilhem/gorkers/blob/master/examples/multiple_workers/multipleworkers.go)
- [Passing Fields](https://github.com/guilhem/gorkers/blob/master/examples/passing_fields/passingfields.go)

## Getting Started

### Import

```go
import (
    "github.com/guilhem/gorkers"
)
```

### Create a worker function 👷

```go
work := func (ctx context.Context, in interface{}, out chan<- interface{}) error {
    // work iteration here
}
```

### Create runner 🚶

```go
runner := gorkers.NewRunner(ctx, work, numberOfWorkers, sizeOfBuffer)
```

- `numberOfWorkers` is the number of parallel workers that can be running at the same time
- `sizeOfBuffer` is the buffer size of input. If stopped, a runner can lose it's buffer.

### Start runner 🏃

```go
if err := runner.Start(); err != nil {
    // error management
}
```

`.Start()` can return an error if `beforeFunc` is in error.

### Send work to worker

```go
runner.Send("Hello World")
```

Send accepts an interface. So send it anything you want.

### Wait for the worker to finish

```go
runner.Wait()
```

`.Wait()` lock any new `.Send()` and block until all jobs are finished.

```go
runner.Close()
```

Use `.Close()` to prevent any new job to be spawn and sending a context cancellation to any worker.

## Working With Multiple Workers

### Passing work form one worker to the next

By using the InFrom method you can tell `workerTwo` to accept output from `workerOne`

```go
runnerOne := gorkers.NewRunner(ctx, work1, 100, 100)
runnerTwo := gorkers.NewRunner(ctx, work2, 100, 100).InFrom(workerOne)

runnerOne.Start()
runnerTwo.Start()

runnerOne.Wait().Stop()
runnerTwo.Wait().Stop()
```

### Accepting output from multiple workers

It is possible to accept output from more than one worker but it is up to you to determine what is coming from which worker. (They will send on the same channel.)

```go
runnerOne := gorkers.gewRunner(ctx, NewMyWorker(), 100, 100)
runnerTwo := gorkers.NewRunner(ctx, NewMyWorkerTwo(), 100, 100)
runnerThree := gorkers.NewRunner(ctx, NewMyWorkerThree(), 100, 100).InFrom(workerOne, workerTwo)
```

## Options

### Timeout

If your workers needs to stop at a deadline or you just need to have a timeout use the SetTimeout or SetDeadline methods. (These must be in place before setting the workers off to work.)

```go
 // Setting a timeout of 2 seconds
 runner.SetWorkerTimeout(2 * time.Second)
```

`.SetWorkerTimeout()` is a timeout for a worker instance to finish.

### Deadline

```go
// Setting a deadline of 4 hours from now
runner.SetDeadline(time.Now().Add(4 \* time.Hour))
```

`.SetDeadline()` is a limit for runner to finish.
