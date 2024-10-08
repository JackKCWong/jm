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
	"os/signal"
	"strings"
	"sync"

	csp "github.com/JackKCWong/chansport"
	"github.com/jmespath/go-jmespath"
)

type app struct {
	Concurrency int
	Verbose     bool
	Input       io.Reader
}

func (app app) Run(ctx context.Context, args []string) error {
	path := args[0]
	jmpath, err := jmespath.Compile(path)
	if err != nil {
		return fmt.Errorf("invalid jmespath: %q", err)
	}

	scanner := bufio.NewScanner(app.Input)
	lines := make(chan string, 100)
	var wg sync.WaitGroup
	if scanner.Scan() {
		firstLine := scanner.Text()
		wg.Add(1)
		if firstLine[:4] == "http" {
			go func() {
				defer wg.Done()
				app.jsonFromRemote(ctx, lines, jmpath)
			}()
		} else {
			go func() {
				defer wg.Done()
				app.jsonFromStdin(lines, jmpath)
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

func main() {
	var app app
	flag.IntVar(&app.Concurrency, "c", 100, "number of concurrent requests, if the input are URLs.")
	flag.BoolVar(&app.Verbose, "v", false, "verbose output")
	flag.Parse()

	if len(flag.Args()) == 0 {
		fmt.Println("usage: jm path < input")
		return
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	app.Input = os.Stdin

	if err := app.Run(ctx, flag.Args()); err != nil {
		log.Println(err)
	}
}

func (app app) jsonFromRemote(ctx context.Context, urls chan string, jmpath *jmespath.JMESPath) {
	res := csp.MapParallel(urls, func(url string) string {
		return app.requestAndSearch(ctx, url, jmpath)
	}, app.Concurrency)

	for r := range res {
		fmt.Print(r)
	}
}

func (app app) requestAndSearch(ctx context.Context, url string, jmpath *jmespath.JMESPath) string {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Sprintf("%s - err: %q", url, err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Sprintf("%s - err: %q", url, err)
	}

	obj := interface{}(nil)
	data, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return fmt.Sprintf("%s - err: %q", url, err)
	}

	err = json.Unmarshal(data, &obj)
	if err != nil {
		return fmt.Sprintf("%s - err: %q", url, err)
	}

	res, err := search(obj, jmpath)
	if err != nil {
		return fmt.Sprintf("%s - err: %q", url, err)
	}

	var sb strings.Builder
	for i := range res {
		if app.Verbose {
			sb.WriteString(fmt.Sprintf("%s:\t%s\n", resp.Request.URL, toJsonStr(res[i])))
		} else {
			sb.WriteString(fmt.Sprintf("%s\n", toJsonStr(res[i])))
		}
	}

	return sb.String()
}

func (app app) jsonFromStdin(jsons chan string, jmpath *jmespath.JMESPath) {
	lineno := -1
	for line := range jsons {
		lineno++
		obj := interface{}(nil)
		err := json.Unmarshal([]byte(line), &obj)
		if err != nil {
			log.Printf("line %d: %q", lineno, err)
			continue
		}

		res, err := search(obj, jmpath)
		if err != nil {
			log.Printf("line %d: %q", lineno, err)
			return
		}

		for i := range res {
			if app.Verbose {
				fmt.Printf("%d:\t%v\n", lineno, toJsonStr(res[i]))
			} else {
				fmt.Printf("%v\n", toJsonStr(res[i]))
			}
		}
	}
}

func search(obj interface{}, jmpath *jmespath.JMESPath) ([]interface{}, error) {
	if objs, ok := obj.([]interface{}); ok {
		var result []interface{}
		for i := range objs {
			res, err := jmpath.Search(objs[i])
			if err != nil {
				return nil, err
			}

			result = append(result, res)
		}

		return result, nil
	}

	res, err := jmpath.Search(obj)
	if err != nil {
		return nil, err
	}

	return []interface{}{res}, nil
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
