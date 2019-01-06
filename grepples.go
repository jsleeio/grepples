package main

import (
  "context"
  "bufio"
  "flag"
  "fmt"
  "log"
  "os"
  "regexp"
  "strings"
  "sort"
  "strconv"
  "time"

  "golang.org/x/crypto/ssh/terminal"
  "golang.org/x/sync/errgroup"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

// Config holds core run configuration
type Config struct {
  Region        *string
  Bucket        *string
  Prefix        *string
  KeyMatch      *string
  ContentMatch  *string
  MaxKeys       *int
  MaxWorkers    *int
  SortByKey     *bool
  TasksTicker   *bool
  ObjectKeys    *bool
  ExtraNewlines *bool
  FitToTTY      *bool
}

// ConfigureFromFlags sets up a config based on commandline flags
func ConfigureFromFlags() *Config {
  return &Config {
		Region:         flag.String("region", "us-west-2", "AWS region to operate in"),
		Bucket:         flag.String("bucket", "", "Name of S3 bucket to operate in"),
		Prefix:         flag.String("prefix", "", "Bucket object base prefix"),
    KeyMatch:       flag.String("key-match", "", "String match on S3 object key"),
    ContentMatch:   flag.String("content-match", "", "String match on S3 object content"),
    MaxKeys:        flag.Int("max-keys", 1000, "Maximum number of keys per page when listing S3 objects"),
    MaxWorkers:     flag.Int("max-workers", 250, "Maximum number of processing workers"),
    SortByKey:      flag.Bool("sort-by-key", true, "Sort output by object key, lexicographically"),
    TasksTicker:    flag.Bool("tasks-ticker", false, "Enable debug logging of task queue length"),
    ObjectKeys:     flag.Bool("object-keys", true, "Include matching object keys in output"),
    ExtraNewlines:  flag.Bool("extra-newlines", true, "Output an extra newline after each object's matches"),
    FitToTTY:       flag.Bool("fit-to-tty", false, "Truncate output lines at $COLUMNS-1 characters"),
  }
}

// Task identifies a discovered piece of work that needs to be processed
// somehow
type Task struct {
  Bucket string
  Key    string
}

// make educated guesses as to whether a key is empty or not; by doing this we
// can skip pointless s3:GetObject API calls
func looksLikeNoContent(key string, length int64) bool {
  if length == 0 {
    return true
  }
  if strings.HasSuffix(key,".bz2") && length == 14 {
    return true
  }
  if strings.HasSuffix(key,".gz") && length == 20 {
    return true
  }
  return false
}

// discoverObjects lists objects matching a prefix in S3, filters out objects
// that look like they're empty, filters out objects whose key does not match a
// regex, and pushes the surviving S3 object keys onto a channel as Tasks
func discoverObjects(config *Config, svc *s3.S3, output chan Task) error {
  defer close(output)
  matchre := regexp.MustCompile(*config.KeyMatch)
	input := &s3.ListObjectsV2Input{
		Bucket:  aws.String(*config.Bucket),
		MaxKeys: aws.Int64(int64(*config.MaxKeys)),
		Prefix:  config.Prefix,
	}
  err := svc.ListObjectsV2Pages(input, func(page *s3.ListObjectsV2Output, last bool) bool {
    n := 0
    for _,obj := range page.Contents {
      if matchre.MatchString(*obj.Key) && ! looksLikeNoContent(*obj.Key, *obj.Size) {
        n++
        output <-Task{Key: *obj.Key, Bucket: *config.Bucket}
      }
    }
    return true
  })
  if err != nil {
    log.Fatalf("error listing objects: %v", err)
  }
  return nil
}

// Result holds the results for a single task
type Result struct {
  Task Task
  Output []string
}

// ByTaskKey implements sort.Interface for []*Result based on the .Task.Key
// field
type ByTaskKey []*Result
func (a ByTaskKey) Len() int           { return len(a) }
func (a ByTaskKey) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByTaskKey) Less(i, j int) bool { return a[i].Task.Key < a[j].Task.Key }

// searchObject retrieves an object from S3 and scans it for matches,
// decompressing if necessary with TransparentExpandingReader. Returns
// a Result 
func searchObject(ctx context.Context, config *Config, svc *s3.S3, id int, task Task) (*Result,error) {
  matchre := regexp.MustCompile(*config.ContentMatch)
  obj,err := svc.GetObject(&s3.GetObjectInput{Bucket: aws.String(task.Bucket), Key: aws.String(task.Key)})
  if err != nil {
    return nil,err
  }
  reader, err := TransparentExpandingReader(task.Key,obj.Body)
  if err != nil {
    log.Printf("worker %d error reading key %s: %v", id, task.Key, err)
    return nil,err
  }
  scanner := bufio.NewScanner(reader)
  result := &Result{Task: task, Output: []string{}}
  for scanner.Scan() {
    text := scanner.Text()
    if matchre.MatchString(text) {
      result.Output = append(result.Output, text)
    }
  }
  return result,nil
}

func ttyWidth() int {
  // try ioctl (TIOCGWINSZ) approach first
  if width,_,err := terminal.GetSize(0); err == nil {
    return width
  }
  // if that fails, try $COLUMNS environment variable
  c := os.Getenv("COLUMNS")
  if width,err := strconv.ParseInt(c,10,16); err == nil && width > 0 {
    return int(width)
  }
  // help. please help
  return 79
}

func leftN(s string, n int) string {
  if n == 0 {
    return s
  }
  l := len(s)
  if n > l {
    return s
  }
  return s[:n]
}

func main() {
  config := ConfigureFromFlags()
  flag.Parse()
	sess, err := session.NewSession(&aws.Config{Region: aws.String(*config.Region)})
  if err != nil {
    log.Fatalf("error setting up S3 client: %v", err)
  }
  svc := s3.New(sess)
  // allow a decent backlog to ensure retrieval of large objects does not block
  // discovery of more objects -- at least not until there is a good queue to
  // process
  tasks := make(chan Task, 10000)
  go func(t chan Task) {
    ticker := time.NewTicker(time.Millisecond*500)
    for range ticker.C {
      if l := len(t); l > 0 {
        log.Printf("tasks remaining: %v", len(t))
      }
    }
  }(tasks)
  workerGroup, ctx := errgroup.WithContext(context.Background())
  output := make(chan *Result)
  workerGroup.Go(func() error { return discoverObjects(config, svc, tasks) })
  for workerID := 0; workerID < *config.MaxWorkers; workerID++ {
    workerGroup.Go(func() error {
      workerID := workerID
      for task := range tasks {
        result,err := searchObject(ctx, config, svc, workerID, task)
        if err != nil {
          log.Printf("search error for %s: %v", task.Key, err)
          return nil
        }
        select {
        case output<-result:
        case <-ctx.Done():
          return ctx.Err()
        }
      }
      return nil
    })
  }
  go func() {
    workerGroup.Wait()
    close(output)
  }()
  totalObjects := 0
  totalMatches := 0
  var results []*Result
  for result := range output {
    totalObjects++
    totalMatches += len(result.Output)
    results = append(results, result)
  }
  sort.Sort(ByTaskKey(results))
  ttyWidth := ttyWidth()
  for _, result := range results {
    if len(result.Output) == 0 {
      continue
    }
    if *config.ObjectKeys {
      fmt.Printf("%s (%d matches):\n", result.Task.Key, len(result.Output))
    }
    for _, line := range result.Output {
      if *config.FitToTTY {
        fmt.Println(leftN(line,ttyWidth))
      } else {
        fmt.Print(line)
      }
    }
    if *config.ExtraNewlines {
      fmt.Println()
    }
    if *config.ObjectKeys {
      fmt.Println()
    }
  }
}
