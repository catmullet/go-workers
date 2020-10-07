![go workers](https://raw.githubusercontent.com/catmullet/go-workers/assets/goworkers.png)
# Usage
* Safely Running groups of workers concurrently or consecutively that require input and output from channels
# Examples
## Standard one go worker setup
```go
  func main() {
    ctx := context.Background()
    worker := goworker.NewWorker(ctx, workerFunction, 100).Work()
    
    for i := 0; i < 10; i++ {
      worker.Send("hello world")
    }
    
    worker.Close()
    err := worker.Wait()
    
    fmt.Println(err)
  }
  
  func workerFunction(w *goworker.Worker) error {
    for in := range w.In() {
      fmt.Println(in)
     }
     return nil
  }
```
## Two workers setup
```go
    func main() {
        ctx := context.Background()
        worker1 := goworker.NewWorker(ctx, workerFunction1, 100).Work()
        worker2 := goworker.NewWorker(ctx, workerFunction2, 100).InFrom(worker1).Work()
    
        for i := 0; i < 10; i++ {
        worker1.Send("hello world")
        }
    
        worker1.Close()
        err := worker1.Wait()
    
        fmt.Println(err)
    
        worker2.Close()
        err = worker2.Wait()
    
        fmt.Println(err)
    }
   
    func workerFunction1(w *goworker.Worker) error {
        for in := range w.In() {
            fmt.Println(fmt.Sprintf("Worker 1, %s", in)
            in = strings.ToTitle(in.(string))
            w.Out(in)
        }
        return nil
    }
  
    func workerFunction2(w *goworker.Worker) error {
        for in := range w.In() {
            fmt.Println(fmt.Sprintf("Worker 2, %s", in)
        }
        return nil
    }
```
## Complete Example
```go
    package main

    import (
	    "context"
	    "fmt"
	    goworker "github.com/catmullet/go-workers"
	    "log"
    )

    type Config struct {
	    ID string
	    Name string
    }

    func main() {
	    ctx := context.Background()

	    config := Config{
		    ID:   "123",
		    Name: "worker3",
	    }
        // Calling .Work() on a new worker creates the number of workers specified and starts 
        // listening on the in channel
	    worker1 := goworker.NewWorker(ctx, workerFunction, 100).Work()
	    worker2 := goworker.NewWorker(ctx, worker2Function, 100).Work()

        // You can pass variables, structs or pointers to the worker in key value form by calling .AddField(key, value)
        // !!This is not meant to replace the in channel!! It is for configuration within the worker function.
	    worker3 := goworker.NewWorker(ctx, worker3Function, 200).
		    AddField("config", config).
        // .InFrom() allows you to listen to the out channel of other workers. It will route all traffic through the 
        // in channel. You can specify multiple workers but it is up to you to decipher incoming traffic. 
		    InFrom(worker1, worker2).
		    Work()

	    for i := 0; i < 10000; i++ {
        // The .Send() method allows you to easily send into the worker through the in channel
		    worker1.Send(fmt.Sprintf("(%d) Hello, from worker1.", i))
		    worker2.Send(fmt.Sprintf("(%d) Hello, from worker2.", i))
	    }

        // Calling .Close() on a worker will close the in channel allowing the worker to finish 
        // up and no longer accept input
	    worker1.Close()
	    worker2.Close()

        // Make sure to call .Wait() on the worker to wait for it to finish up what is left in the channel buffer
        // Wait() will return an error from the function if there is one.
	    err := worker1.Wait()
	    if err != nil {
		    log.Fatal(err)
	    }

	    err = worker2.Wait()
	    if err != nil {
		    log.Fatal(err)
	    }

	    worker3.Close()
	    err = worker3.Wait()
    }

    // These are examples of worker function.  They must accept a worker, return an error and for loop over 
    // the .In() channel
    func workerFunction(w *goworker.Worker) (err error) {
	    for in := range w.In() {
		    w.Out(in)
	    }
	    return nil
    }

    func worker2Function(w *goworker.Worker) (err error) {
	    for in := range w.In() {
		    w.Out(in)
	    }
	    return nil
    }

    func worker3Function(w *goworker.Worker) (err error) {

    // This is how to get a struct out of the key value store.
	    config := Config{}
	    w.GetFieldObject("config", &config)

	    for in := range w.In() {
		    fmt.Println(in, fmt.Sprintf(" %s says hi also.", config.Name))
	    }
	    return nil
    }
```
