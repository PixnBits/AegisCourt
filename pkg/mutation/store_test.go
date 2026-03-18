package mutation

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGenerateMutationID(t *testing.T) {
	id := GenerateMutationID("test-123")
	if id == "" {
		t.Error("empty mutation ID")
	}
	if len(id) < 10 {
		t.Errorf("mutation ID too short: %s", id)
	}
	if id[:4] != "mut-" {
		t.Errorf("mutation ID should start with mut-, got: %s", id)
	}
}

func TestSaveLoadMutation(t *testing.T) {
	// Use temp dir
	tmpDir := t.TempDir()
	os.Setenv("AEGISCOURT_HOME", tmpDir)
	defer os.Unsetenv("AEGISCOURT_HOME")

	m := &Mutation{
		ID:         "mut-test-001",
		ProposalID: "p-001",
		Type:       "add-tool",
		Title:      "Test mutation",
		Status:     StatusApplied,
	}

	if err := SaveMutation(m); err != nil {
		t.Fatal(err)
	}

	loaded, err := LoadMutation("mut-test-001")
	if err != nil {
		t.Fatal(err)
	}
	if loaded.ID != m.ID || loaded.Status != StatusApplied {
		t.Errorf("loaded mutation mismatch: %+v", loaded)
	}
}

func TestListMutations(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("AEGISCOURT_HOME", tmpDir)
	defer os.Unsetenv("AEGISCOURT_HOME")

	for _, id := range []string{"mut-a", "mut-b"} {
		SaveMutation(&Mutation{
			ID:     id,
			Status: StatusApplied,
		})
	}

	muts, err := ListMutations()
	if err != nil {
		t.Fatal(err)
	}
	if len(muts) != 2 {
		t.Errorf("expected 2 mutations, got %d", len(muts))
	}
}

func TestCreateRestoreSnapshot(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("AEGISCOURT_HOME", tmpDir)
	defer os.Unsetenv("AEGISCOURT_HOME")

	// Create a test file to snapshot
	testFile := filepath.Join(tmpDir, "test.txt")
	os.WriteFile(testFile, []byte("original"), 0600)

	snapPath, err := CreateSnapshot("test-snap")
	if err != nil {
		t.Fatal(err)
	}
	if snapPath == "" {
		t.Fatal("empty snapshot path")
	}

	// Modify the file
	os.WriteFile(testFile, []byte("modified"), 0600)

	// Restore
	if err := RestoreSnapshot(snapPath); err != nil {
		t.Fatal(err)
	}

	data, _ := os.ReadFile(testFile)
	if string(data) != "original" {
		t.Errorf("expected 'original' after restore, got %q", string(data))
	}
}
