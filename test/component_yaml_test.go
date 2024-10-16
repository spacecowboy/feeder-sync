package test

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"
	"gopkg.in/yaml.v2"
)

type TestCase struct {
	Name     string   `yaml:"name"`
	Request  Request  `yaml:"request"`
	Response Response `yaml:"response"`
}

type Request struct {
	Method  string            `yaml:"method"`
	Path    string            `yaml:"path"`
	Headers map[string]string `yaml:"headers"`
	Body    string            `yaml:"body"`
}

type Response struct {
	Status  int               `yaml:"status"`
	Headers map[string]string `yaml:"headers"`
	Body    string            `yaml:"body"`
}

type YamlTestSuite struct {
	suite.Suite
	Tests    []TestCase
	FilePath string
}

// SetupSuite loads the test cases from the YAML file
func (suite *YamlTestSuite) SetupSuite() {
	data, err := os.ReadFile(suite.FilePath)
	if err != nil {
		suite.T().Fatalf("Failed to read YAML file: %s", err)
	}

	err = yaml.Unmarshal(data, &suite.Tests)
	if err != nil {
		suite.T().Fatalf("Failed to unmarshal YAML: %s", err)
	}
}

// SetupTests
func (suite *YamlTestSuite) SetupTest() {
	// Do nothing
}

// TearDownTests
func (suite *YamlTestSuite) TearDownTest() {
	// Do nothing
}

// TearDownSuite
func (suite *YamlTestSuite) TearDownSuite() {
	// Do nothing
}

// TestCases runs the test cases
func (suite *YamlTestSuite) TestCases() {
	baseUrl := fmt.Sprintf("http://%s", listenAddress)

	for _, testCase := range suite.Tests {
		suite.Run(testCase.Name, func() {
			t := suite.T()

			req, err := http.NewRequest(
				testCase.Request.Method,
				fmt.Sprintf("%s%s", baseUrl, testCase.Request.Path),
				strings.NewReader(testCase.Request.Body),
			)
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}

			for k, v := range testCase.Request.Headers {
				req.Header.Add(k, v)
			}

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatalf("Failed to send request: %v", err)
			}
			defer resp.Body.Close()

			suite.Equal(testCase.Response.Status, resp.StatusCode)

			for k, v := range testCase.Response.Headers {
				suite.Equal(v, resp.Header.Get(k))
			}

			if testCase.Response.Body == "" {
				// No body to check
				return
			}

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("Failed to read response body: %v", err)
			}

			suite.Equal(strings.TrimSpace(testCase.Response.Body), strings.TrimSpace(string(body)))
		})
	}
}

// Run all test suites
func TestYamlSuites(t *testing.T) {
	files, err := filepath.Glob("*.yaml")
	if err != nil {
		t.Fatalf("Failed to find YAML files: %v", err)
	}

	for _, filePath := range files {
		suite.Run(t, &YamlTestSuite{FilePath: filePath})
	}
}
