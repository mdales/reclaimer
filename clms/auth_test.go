package clms

import (
	"testing"
)

func TestLoadingCLMSAPIToken(t *testing.T) {
	token, err := LoadAPIKey("testdata/exampleapi.key")
	if nil != err {
		t.Errorf("Failed to load key: %v", err)
	}
	if "" == token.ClientID {
		t.Errorf("Didn't get token when expected.")
	}
}
