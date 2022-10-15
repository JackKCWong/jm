package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sync"

	"github.com/jmespath/go-jmespath"
	"golang.org/x/sync/semaphore"
)

func main() {
	fConcurrency := flag.Int("c", 100, "number of concurrent requests, if the input are URLs.")
	flag.Parse()

	if err := runMain(os.Stdin, *fConcurrency, flag.Arg(0)); err != nil {
		log.Println(err)
	}
}

func runMain(input io.Reader, concurrency int, path string) error {
	jmpath, err := jmespath.Compile(path)
	if err != nil {
		return fmt.Errorf("invalid jmespath: %q", err)
	}

	scanner := bufio.NewScanner(input)
	lines := make(chan string, 100)
	var wg sync.WaitGroup
	if scanner.Scan() {
		firstLine := scanner.Text()
		wg.Add(1)
		if firstLine[:4] == "http" {
			go func() {
				defer wg.Done()
				jsonFromRemote(lines, jmpath, concurrency)
			}()
		} else {
			go func() {
				defer wg.Done()
				jsonFromStdin(lines, jmpath)
			}()
		}

		lines <- firstLine

		for scanner.Scan() {
			lines <- scanner.Text()
		}

		close(lines)

		wg.Wait()
	}

	return nil
}

func jsonFromRemote(urls chan string, jmpath *jmespath.JMESPath, concurrency int) {
	var wg sync.WaitGroup
	sem := semaphore.NewWeighted(int64(concurrency))
	lineno := -1
	for url := range urls {
		wg.Add(1)
		lineno++
		if err := sem.Acquire(context.TODO(), 1); err != nil {
			log.Printf("err: %q", err)
			return
		}
		go func(url string, lineno int) {
			defer wg.Done()
			defer sem.Release(1)
			err := requestAndSearch(url, jmpath)
			if err != nil {
				log.Printf("line %d: %q", lineno, err)
			}
		}(url, lineno)
	}

	wg.Wait()
}

func requestAndSearch(url string, jmpath *jmespath.JMESPath) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}

	obj := interface{}(nil)
	data, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return err
	}

	err = json.Unmarshal(data, &obj)
	if err != nil {
		return err
	}

	res, err := jmpath.Search(obj)
	if err != nil {
		return err
	}

	fmt.Printf("%s:  %s\n", resp.Request.URL, toJsonStr(res))

	return nil
}

func jsonFromStdin(jsons chan string, jmpath *jmespath.JMESPath) {
	lineno := -1
	for line := range jsons {
		lineno++
		obj := interface{}(nil)
		err := json.Unmarshal([]byte(line), &obj)
		if err != nil {
			log.Printf("line %d: %q", lineno, err)
			continue
		}

		res, err := jmpath.Search(obj)
		if err != nil {
			log.Printf("line %d: %q", lineno, err)
		}

		fmt.Printf("%d:  %v\n", lineno, toJsonStr(res))
	}
}

func toJsonStr(v interface{}) string {
	out := ""
	if v != nil {
		ret, err := json.Marshal(&v)
		if err != nil {
			out = err.Error()
		} else {
			out = string(ret)
		}
	}

	return out
}
