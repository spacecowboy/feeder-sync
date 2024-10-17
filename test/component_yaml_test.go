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

func (suite *YamlTestSuite) replacevariables(input string) string {
	replacements := []string{}
	for k, v := range suite.Variables {
		replacements = append(replacements, fmt.Sprintf("{{%s}}", k), v)
	}
	replacer := strings.NewReplacer(replacements...)
	return replacer.Replace(input)
}

// TestCases runs the test cases
func (suite *YamlTestSuite) TestCases() {
	baseUrl := fmt.Sprintf("http://%s", listenAddress)

	for _, testCase := range suite.Tests {
		suite.Run(testCase.Name, func() {
			t := suite.T()

			// Create a copy of the testCase to avoid modifying the original
			tc := testCase

			// Replace variables in the request body
			tc.Request.Body = suite.replacevariables(tc.Request.Body)

			// Replace variables in the headers
			for k, v := range tc.Request.Headers {
				tc.Request.Headers[k] = suite.replacevariables(v)
			}

			// Ensure the body is a valid JSON
			if tc.Request.Body != "" {
				err := json.Unmarshal([]byte(tc.Request.Body), &map[string]interface{}{})
				require.NoErrorf(t, err, "Body was %s", tc.Request.Body)
			}

			req, err := http.NewRequest(
				tc.Request.Method,
				fmt.Sprintf("%s%s", baseUrl, tc.Request.Path),
				strings.NewReader(tc.Request.Body),
			)
			require.NoError(t, err, "Failed to create request")

			for k, v := range tc.Request.Headers {
				req.Header.Add(k, v)
			}

			resp, err := http.DefaultClient.Do(req)
			require.NoError(t, err, "Failed to send request")
			defer resp.Body.Close()

			respBody, err := io.ReadAll(resp.Body)
			require.NoError(t, err, "Failed to read response body")

			// Check response status
			suite.Equal(tc.Response.Status, resp.StatusCode)

			// Check response headers
			for k, v := range tc.Response.Headers {
				suite.Equal(v, resp.Header.Get(k), "Header did not match, body: %s", string(respBody))
			}

			if tc.Response.Body != "" {
				var expectedBody, actualBody map[string]interface{}
				err = json.Unmarshal([]byte(tc.Response.Body), &expectedBody)
				require.NoError(t, err, "Failed to unmarshal expected response body")

				err = json.Unmarshal(respBody, &actualBody)
				require.NoError(t, err, "Failed to unmarshal actual response body")

				suite.Equal(expectedBody, actualBody)
			}

			// Check response body types
			for jsonPath, expectedType := range tc.Response.BodyTypes {
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
			for varName, jsonPath := range tc.Extract {
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
