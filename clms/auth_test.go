package clms

import (
	"fmt"
	"os"
	"testing"
)

func TestLoadingCLMSAPIToken(t *testing.T) {
	cwd, err := os.Getwd()
	if nil != err {
		t.Fatalf("Failed to get CWD: %v", err)
	}
	fmt.Printf("cwd: %s\n", cwd)

	token, err := LoadAPIKey("testdata/exampleapi.key")
	if nil != err {
		t.Errorf("Failed to load key: %v", err)
	}
	if "" == token.ClientID {
		t.Errorf("Didn't get token when expected.")
	}
}
