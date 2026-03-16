package auditstore

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

// CreateSnapshot creates a snapshot archive.
func (as *AuditStore) CreateSnapshot(name string, enterprise bool) (string, error) {
	snapshotDir := filepath.Join(os.Getenv("HOME"), ".aegiscourt", "snapshots")
	os.MkdirAll(snapshotDir, 0700)

	tarPath := filepath.Join(snapshotDir, fmt.Sprintf("%s.tar.gz", name))

	file, err := os.Create(tarPath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	gw := gzip.NewWriter(file)
	defer gw.Close()

	tw := tar.NewWriter(gw)
	defer tw.Close()

	// Add audit log
	logPath := filepath.Join(os.Getenv("HOME"), ".aegiscourt", "audit.log.jsonl")
	as.log.ExportJSONL(logPath)
	if err := addFileToTar(tw, logPath, "audit.log.jsonl"); err != nil {
		return "", err
	}

	// Add kernel hash (stub)
	kernelHashPath := filepath.Join(os.Getenv("HOME"), ".aegiscourt", "kernel.sig")
	if _, err := os.Stat(kernelHashPath); err == nil {
		addFileToTar(tw, kernelHashPath, "kernel.sig")
	}

	// Add constitution
	constitutionPath := filepath.Join("docs", "constitution.md")
	if _, err := os.Stat(constitutionPath); err == nil {
		addFileToTar(tw, constitutionPath, "constitution.md")
	}

	// SBOM stub
	sbomContent := `{"sbom": "stub"}`
	addContentToTar(tw, sbomContent, "sbom.json")

	if enterprise {
		// NIST mapping stub
		nistContent := `{"nist_mapping": "stub"}`
		addContentToTar(tw, nistContent, "nist-mapping.json")
	}

	// Merkle root
	rootContent := fmt.Sprintf(`{"merkle_root": "%x"}`, as.GetRootHash())
	addContentToTar(tw, rootContent, "merkle-root.json")

	return tarPath, nil
}

// addFileToTar adds a file to the tar archive.
func addFileToTar(tw *tar.Writer, filePath, nameInTar string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return err
	}

	header := &tar.Header{
		Name:    nameInTar,
		Size:    stat.Size(),
		Mode:    int64(stat.Mode()),
		ModTime: time.Now(),
	}

	if err := tw.WriteHeader(header); err != nil {
		return err
	}

	_, err = io.Copy(tw, file)
	return err
}

// addContentToTar adds string content to the tar archive.
func addContentToTar(tw *tar.Writer, content, nameInTar string) error {
	header := &tar.Header{
		Name:    nameInTar,
		Size:    int64(len(content)),
		Mode:    0600,
		ModTime: time.Now(),
	}

	if err := tw.WriteHeader(header); err != nil {
		return err
	}

	_, err := tw.Write([]byte(content))
	return err
}
