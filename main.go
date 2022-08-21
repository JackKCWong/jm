package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sync"

	"github.com/jmespath/go-jmespath"
)

func main() {
	flag.Parse()

	if err := runMain(os.Stdin, flag.Arg(0)); err != nil {
		log.Println(err)
	}
}

func runMain(input io.Reader, path string) error {
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
				jsonFromRemote(lines, jmpath)
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

func jsonFromRemote(urls chan string, jmpath *jmespath.JMESPath) {
	var wg sync.WaitGroup
	lineno := -1
	for url := range urls {
		wg.Add(1)
		lineno++
		go func(url string, lineno int) {
			defer wg.Done()
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

	fmt.Printf("%s:  %v\n", resp.Request.URL, res)

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

		fmt.Printf("%d:  %v\n", lineno, res)
	}
}
