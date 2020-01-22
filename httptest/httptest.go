// Package httptest implements types and functions for testing WAFs.
package httptest

import (
	"crypto/rand"
	"fmt"
	"log"
	"net/http"
	"os"
	"path"
	"strings"
	"time"

	"github.com/jreisinger/waf-tester/yaml"
)

// GetTests returns the list of available tests.
func GetTests(path string) ([]Test, error) {
	var tests []Test

	// Check directory exists.
	if _, err := os.Stat(path); err != nil {
		return tests, err
	}

	yamls := yaml.ParseFiles(path)
	for _, yaml := range yamls {
		for _, test := range yaml.Tests {
			t := &Test{
				Title:               test.Title,
				Desc:                test.Desc,
				File:                test.File,
				Method:              test.Stages[0].Stage.Input.Method,
				Path:                test.Stages[0].Stage.Input.URI,
				Headers:             test.Stages[0].Stage.Input.Headers,
				Data:                test.Stages[0].Stage.Input.Data,
				ExpectedStatusCodes: test.Stages[0].Stage.Output.Status,
			}
			t.setFields()
			tests = append(tests, *t)
		}
	}

	return tests, nil
}

// https://yourbasic.org/golang/generate-uuid-guid/
func genUUID() string {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		log.Fatal(err)
	}
	uuid := fmt.Sprintf("%x-%x-%x-%x-%x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
	return uuid
}

func (t *Test) setFields() {
	t.ID = genUUID()
	t.Headers["waf-tester-id"] = t.ID
	if t.Desc == "" {
		t.Desc = "No test description"
	}
	if len(t.ExpectedStatusCodes) == 0 {
		t.ExpectedStatusCodes = append(t.ExpectedStatusCodes, 403)
	}
	if t.Method == "" {
		t.Method = "XXX"
	}
}

func intInSlice(n int, slice []int) bool {
	for _, m := range slice {
		if n == m {
			return true
		}
	}
	return false
}

func (t *Test) setTestStatusField() {
	switch {
	case intInSlice(t.StatusCode, t.ExpectedStatusCodes):
		t.TestStatus = "OK"
	case intInSlice(t.StatusCode, []int{0}):
		t.TestStatus = "ERR"
	default:
		t.TestStatus = "FAIL"
	}
}

// Execute executes a Test. It fills in some of the Test fields (like URL, StatusCode).
func (t *Test) Execute(host string) {
	t.URL = "http" + "://" + path.Join(host, t.Path)

	defer t.setTestStatusField()

	data := strings.Join(t.Data, "")
	req, err := http.NewRequest(t.Method, t.URL, strings.NewReader(data))
	if err != nil {
		t.Err = err
		return
	}

	for k, v := range t.Headers {
		req.Header.Set(k, v)
	}

	client := &http.Client{Timeout: time.Second * 10}

	resp, err := client.Do(req)
	if err != nil {
		t.Err = err
		return
	}
	defer resp.Body.Close()

	t.StatusCode = resp.StatusCode
	t.Status = resp.Status
}
