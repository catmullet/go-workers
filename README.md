![go workers](https://raw.githubusercontent.com/catmullet/go-workers/assets/goworkers.png)
# Usage
* Safely Running groups of workers concurrently or consecutively that require input and output from channels
# Examples
## Standard one go worker setup
```go
  func main() {
    ctx := context.Background()
    worker := goworker.NewWorker(ctx, workerFunction, 100).Work()
    
    worker.Send("hello world")
    
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
    worker1 := goworker.NewWorker(ctx, workerFunction, 100).Work()
    worker2 := goworker.NewWorker(ctx, workerFunction, 100).InFrom(worker1).Work()
    
    worker1.Send("hello world")
    
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
