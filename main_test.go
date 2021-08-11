package gcedns

import (
	"context"
	"strings"
	"testing"
)

type CheckOperationTestData struct {
	logSnippet     []byte
	expectedResult string
	testcaseStatus bool
}

func TestCheckOperation(t *testing.T) {

	test_data := []CheckOperationTestData{
		{
			logSnippet:     []byte(""),
			expectedResult: "checkOperation received null data",
			testcaseStatus: true,
		},
		{
			logSnippet:     []byte("invalid data"),
			expectedResult: "No VM info received.",
			testcaseStatus: false,
		},
	}

	for _, data := range test_data {
		result := checkOperation(data.logSnippet, context.Background())

		if data.testcaseStatus {
			if !strings.Contains(result, data.expectedResult) {
				t.Errorf("FAILED: Error occured: got %v expected %v\n", result, data.expectedResult)
			} else {
				t.Logf("PASSED: No error: got %v expected %v\n", result, data.expectedResult)
			}
		} else {
			if !strings.Contains(result, data.expectedResult) {
				t.Errorf("FAILED: Error occured: got %v expected %v\n", result, data.expectedResult)
			} else {
				t.Logf("PASSED: No error: got %v expected %v\n", result, data.expectedResult)
			}
		}
	}
}
