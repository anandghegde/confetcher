package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"bufio"
	"sync"
	"strconv"
)
type Run struct {
	n int
	max int
	work chan func() error
	done chan error
	err chan error
	wg sync.WaitGroup
}
type Errors []error

func (errs Errors) Error() string {
	switch len(errs) {
	case 0:
		return "no error"
	case 1:
		return errs[0].Error()
	}
	return fmt.Sprintf("%s (and %d more)", errs[0].Error(), len(errs) - 1)
}
func NewRun(maxPar int) *Run {
	r := &Run{
		max: maxPar,
		work: make(chan func() error),
		done: make(chan error),
		err: make(chan error),
	}
	go func() {
		var errs Errors
		for e := range r.done {
			if e != nil {
				errs = append(errs, e)
			}
		}
		// TODO sort errors by original order of Do request?
		if len(errs) > 0 {
			r.err <- errs
		} else {
			r.err <- nil
		}
	}()
	return r
}
func (r *Run) Do(f func() error) {
	if r.n < r.max {
		r.wg.Add(1)
		go func(){
			for f := range r.work {
				r.done <- f()
			}
			r.wg.Done()
		}()
	}
	r.work <- f
	r.n++
}

func (r *Run) Wait() error {
	close(r.work)
	r.wg.Wait()
	close(r.done)
	return <-r.err
}



var mFlag = flag.Int("m", 3, "-m n")
var inputflag = flag.String("i", "", "-i input_file")


func main() {
	flag.Parse()
	inputFile := *inputflag
	maxWorkers := *mFlag
	if (inputFile == "" ){
		fmt.Fprintf(os.Stderr, "usage: %s -i inputFile -m <max_no_of_workers>\n", os.Args[0])
		os.Exit(1)
	}
	if(maxWorkers==3){
		fmt.Println("Taking the numbers of workers as 3")
	}
	openedFile, err := os.Open(inputFile)
	if err != nil {
		return
	}
	reader := bufio.NewReader(openedFile)
	var urls []string
	for {
		part, prefix, err := reader.ReadLine()
		if err != nil {
			break
		}
		if !prefix {
			urls = append(urls, string(part))
		}
	}
	noOfWorkers:=maxWorkers
	r := NewRun(noOfWorkers-1)

	for i := range urls {
		url := urls[i]
		fmt.Printf("Now fetching url - %s\n", url)
		outputFile := "file" + strconv.Itoa(i)
		r.Do(func() error {
			response, err := http.Get(url)
			if err != nil {
				return nil
			}
			responseString, err := ioutil.ReadAll(response.Body)
			response.Body.Close()
			err = ioutil.WriteFile(outputFile, responseString, 0644)
			fmt.Printf("Finished fetching url%s\n", url)
			return err
		})
	}
	err = r.Wait()
	fmt.Printf("final err: %v", err)
}
