package test

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/tidwall/gjson"
	"gopkg.in/yaml.v2"
)

type TestCase struct {
	Name     string            `yaml:"name"`
	Request  Request           `yaml:"request"`
	Response Response          `yaml:"response"`
	Extract  map[string]string `yaml:"extract"`
}

type Request struct {
	Method  string            `yaml:"method"`
	Path    string            `yaml:"path"`
	Headers map[string]string `yaml:"headers"`
	Body    string            `yaml:"body"`
}

type Response struct {
	Status    int               `yaml:"status"`
	Headers   map[string]string `yaml:"headers"`
	Body      string            `yaml:"body"`
	BodyTypes map[string]string `yaml:"bodyTypes"`
}

type YamlTestSuite struct {
	suite.Suite
	Tests     []TestCase
	FilePath  string
	Variables map[string]string
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

	suite.Variables = make(map[string]string)
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

			// Replace variables in the request body
			body := testCase.Request.Body
			for k, v := range suite.Variables {
				body = strings.ReplaceAll(body, fmt.Sprintf("{{%s}}", k), v)
			}

			// Replace variables in the headers
			for k, v := range testCase.Request.Headers {
				for varName, varValue := range suite.Variables {
					testCase.Request.Headers[k] = strings.ReplaceAll(v, fmt.Sprintf("{{%s}}", varName), varValue)
				}
			}

			// Ensure the body is a valid JSON
			if body != "" {
				err := json.Unmarshal([]byte(body), &map[string]interface{}{})
				require.NoErrorf(t, err, "Body was %s", body)
			}

			req, err := http.NewRequest(
				testCase.Request.Method,
				fmt.Sprintf("%s%s", baseUrl, testCase.Request.Path),
				strings.NewReader(body),
			)
			require.NoError(t, err, "Failed to create request")

			for k, v := range testCase.Request.Headers {
				req.Header.Add(k, v)
			}

			resp, err := http.DefaultClient.Do(req)
			require.NoError(t, err, "Failed to send request")
			defer resp.Body.Close()

			respBody, err := io.ReadAll(resp.Body)
			require.NoError(t, err, "Failed to read response body")

			// Check response status
			suite.Equal(testCase.Response.Status, resp.StatusCode)

			// Check response headers
			for k, v := range testCase.Response.Headers {
				suite.Equal(v, resp.Header.Get(k), "Header did not match, body: %s", string(respBody))
			}

			if testCase.Response.Body != "" {
				var expectedBody, actualBody map[string]interface{}
				err = json.Unmarshal([]byte(testCase.Response.Body), &expectedBody)
				require.NoError(t, err, "Failed to unmarshal expected response body")

				err = json.Unmarshal(respBody, &actualBody)
				require.NoError(t, err, "Failed to unmarshal actual response body")

				suite.Equal(expectedBody, actualBody)
			}

			// Check response body types
			for jsonPath, expectedType := range testCase.Response.BodyTypes {
				value := gjson.Get(string(respBody), jsonPath)
				switch expectedType {
				case "string":
					suite.Equal(gjson.String, value.Type, "Expected string type for %s", jsonPath)
				case "number":
					suite.Equal(gjson.Number, value.Type, "Expected number type for %s", jsonPath)
				case "bool":
					suite.Equal(gjson.True, value.Type, "Expected bool type for %s", jsonPath)
				default:
					t.Fatalf("Unsupported type: %s", expectedType)
				}
			}

			// Extract variables from the response body
			for varName, jsonPath := range testCase.Extract {
				// Extract value from the response body
				value := gjson.Get(string(respBody), jsonPath)

				suite.Variables[varName] = value.String()
			}
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
