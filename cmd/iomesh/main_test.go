package main

import "testing"

func TestCmdMeshStreams_CreateRequiresYes(t *testing.T) {
	if code := cmdMeshStreams([]string{"--create"}); code != 2 {
		t.Fatalf("missing --yes exit=%d want 2", code)
	}
}

func TestCmdMeshStreams_CreateIncompatibleWithDeleteAndMessages(t *testing.T) {
	if code := cmdMeshStreams([]string{"--create", "--yes", "--delete", "--name", "X"}); code != 2 {
		t.Fatalf("create+delete exit=%d want 2", code)
	}
	if code := cmdMeshStreams([]string{"--create", "--yes", "--messages", "--name", "X"}); code != 2 {
		t.Fatalf("create+messages exit=%d want 2", code)
	}
}
