package clms

import (
	"encoding/json"
	"testing"
)

func TestEncodeDatasetRequest(t *testing.T) {
	// taken from the CLMS API examples in their docs, except fixed quotes from python
	raw := `
{"Datasets": [
    {
     "DatasetID": "d30959acacf84e418c658ecaf6673ebf",
     "FileID": "7b2c4ca3-749c-4c3d-b4cd-a848e7465175"
    },
    {
     "DatasetID": "d30959acacf84e418c658ecaf6673ebf",
     "FileID": "345b7d52-fa71-43da-a182-dca7a74a16e2"
    }
]}
`
	var result CLMSPrepackagedDataRequest
	err := json.Unmarshal([]byte(raw), &result)
	if nil != err {
		t.Errorf("Expected no error, got %v", err)
	} else {
		if len(result.Datasets) != 2 {
			t.Errorf("Expected two datasets, got %d", len(result.Datasets))
		} else {
			for idx, dataset := range result.Datasets {
				if "" == dataset.FileID {
					t.Errorf("Dataset %d had empty fileID", idx)
				}
				if "" == dataset.DatasetID {
					t.Errorf("Dataset %d had tempty datasetID", idx)
				}
			}
		}
	}
}
