package mutation

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/pixnbits/aegiscourt/pkg/keys"
)

func mutationsDir() (string, error) {
	dir, err := keys.AegisCourtDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "mutations"), nil
}

func snapshotsDir() (string, error) {
	dir, err := keys.AegisCourtDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "snapshots"), nil
}

func SaveMutation(m *Mutation) error {
	dir, err := mutationsDir()
	if err != nil {
		return err
	}
	if err := keys.EnsureDir(dir); err != nil {
		return err
	}
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal mutation: %w", err)
	}
	path := filepath.Join(dir, m.ID+".json")
	return os.WriteFile(path, data, 0600)
}

func LoadMutation(id string) (*Mutation, error) {
	dir, err := mutationsDir()
	if err != nil {
		return nil, err
	}
	path := filepath.Join(dir, id+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("mutation not found: %s", id)
	}
	var m Mutation
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("corrupt mutation: %w", err)
	}
	return &m, nil
}

func ListMutations() ([]*Mutation, error) {
	dir, err := mutationsDir()
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var mutations []*Mutation
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		id := entry.Name()[:len(entry.Name())-5]
		m, err := LoadMutation(id)
		if err == nil {
			mutations = append(mutations, m)
		}
	}
	sort.Slice(mutations, func(i, j int) bool {
		return mutations[i].AppliedAt.Before(mutations[j].AppliedAt)
	})
	return mutations, nil
}

func LastAppliedMutation() (*Mutation, error) {
	muts, err := ListMutations()
	if err != nil {
		return nil, err
	}
	for i := len(muts) - 1; i >= 0; i-- {
		if muts[i].Status == StatusApplied {
			return muts[i], nil
		}
	}
	return nil, nil
}

func CreateSnapshot(mutationID string) (string, error) {
	baseDir, err := keys.AegisCourtDir()
	if err != nil {
		return "", err
	}
	snapDir, err := snapshotsDir()
	if err != nil {
		return "", err
	}
	if err := keys.EnsureDir(snapDir); err != nil {
		return "", err
	}

	snapPath := filepath.Join(snapDir, mutationID+".tar.gz")
	f, err := os.Create(snapPath)
	if err != nil {
		return "", fmt.Errorf("failed to create snapshot file: %w", err)
	}
	defer f.Close()

	gw := gzip.NewWriter(f)
	defer gw.Close()
	tw := tar.NewWriter(gw)
	defer tw.Close()

	err = filepath.Walk(baseDir, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, _ := filepath.Rel(baseDir, path)
		if strings.HasPrefix(rel, "snapshots") {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasPrefix(rel, "mutations") {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		header.Name = rel

		if err := tw.WriteHeader(header); err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()
		_, err = io.Copy(tw, file)
		return err
	})

	if err != nil {
		os.Remove(snapPath)
		return "", fmt.Errorf("failed to create snapshot: %w", err)
	}

	return snapPath, nil
}

func RestoreSnapshot(snapshotPath string) error {
	baseDir, err := keys.AegisCourtDir()
	if err != nil {
		return err
	}

	f, err := os.Open(snapshotPath)
	if err != nil {
		return fmt.Errorf("snapshot not found: %w", err)
	}
	defer f.Close()

	gr, err := gzip.NewReader(f)
	if err != nil {
		return fmt.Errorf("corrupt snapshot: %w", err)
	}
	defer gr.Close()

	tr := tar.NewReader(gr)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("corrupt snapshot entry: %w", err)
		}

		target := filepath.Join(baseDir, header.Name)
		if !strings.HasPrefix(filepath.Clean(target), filepath.Clean(baseDir)) {
			return fmt.Errorf("invalid path in snapshot: %s", header.Name)
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, os.FileMode(header.Mode)); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0700); err != nil {
				return err
			}
			out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(header.Mode))
			if err != nil {
				return err
			}
			if _, err := io.Copy(out, io.LimitReader(tr, 100*1024*1024)); err != nil {
				out.Close()
				return err
			}
			out.Close()
		}
	}

	return nil
}

func GenerateMutationID(proposalID string) string {
	return fmt.Sprintf("mut-%s-%s", proposalID, time.Now().UTC().Format("20060102T150405"))
}
