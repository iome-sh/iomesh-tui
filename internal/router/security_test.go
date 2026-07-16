package router

import (
	"strings"
	"testing"
)

func TestNew_RejectsFileScheme(t *testing.T) {
	_, err := New([]ModelConfig{{
		Name: "x", BaseURL: "file:///tmp", ModelID: "m", APIKey: "k", Priority: 1, MaxContext: 1000,
	}}, "x")
	if err == nil || !strings.Contains(err.Error(), "scheme") {
		t.Fatalf("err=%v", err)
	}
}

func TestAPIError_RedactsBodyViaCheckStatus(t *testing.T) {
	err := checkStatus(401, []byte(`{"error":"Bearer sk-abcdefghijklmnopqrstuvwxyz"}`))
	if err == nil {
		t.Fatal()
	}
	if strings.Contains(err.Error(), "sk-abcdefgh") {
		t.Fatalf("leaked key: %v", err)
	}
}
