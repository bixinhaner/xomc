package agentruntime

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"path"
	"strings"
	"sync"

	"github.com/omcgo/omcgo/internal/agentruntime/handbookasset"
)

const (
	handbookArchiveFormat       = "tar+gzip"
	handbookContentRoot         = "references"
	handbookChunkBytes          = 128 * 1024
	maxHandbookArchiveBytes     = 8 * 1024 * 1024
	maxHandbookUncompressedSize = 64 * 1024 * 1024
	maxHandbookFiles            = 5000
)

type HandbookPackageManifest struct {
	SchemaVersion   string `json:"schemaVersion"`
	CatalogVersion  string `json:"catalogVersion"`
	HandbookDigest  string `json:"handbookDigest"`
	TotalOperations int    `json:"totalOperations"`
	ManifestPath    string `json:"manifestPath"`
	ChunkPath       string `json:"chunkPathTemplate"`
	ArchiveFormat   string `json:"archiveFormat"`
	ArchiveBytes    int    `json:"archiveBytes"`
	ChunkBytes      int    `json:"chunkBytes"`
	TotalChunks     int    `json:"totalChunks"`
	ContentRoot     string `json:"contentRoot"`
}

type HandbookPackageChunk struct {
	HandbookDigest string `json:"handbookDigest"`
	Index          int    `json:"index"`
	TotalChunks    int    `json:"totalChunks"`
	Bytes          int    `json:"bytes"`
	Encoding       string `json:"encoding"`
	Data           string `json:"data"`
}

type handbookPackage struct {
	archive  []byte
	manifest HandbookPackageManifest
}

var (
	embeddedHandbookOnce sync.Once
	embeddedHandbook     *handbookPackage
	embeddedHandbookErr  error
)

func loadEmbeddedHandbookPackage() (*handbookPackage, error) {
	embeddedHandbookOnce.Do(func() {
		embeddedHandbook, embeddedHandbookErr = newHandbookPackage(handbookasset.Archive())
	})
	return embeddedHandbook, embeddedHandbookErr
}

func newHandbookPackage(archive []byte) (*handbookPackage, error) {
	if len(archive) == 0 {
		return nil, fmt.Errorf("embedded handbook package is empty")
	}
	if len(archive) > maxHandbookArchiveBytes {
		return nil, fmt.Errorf("embedded handbook package exceeds %d bytes", maxHandbookArchiveBytes)
	}
	gzipReader, err := gzip.NewReader(bytes.NewReader(archive))
	if err != nil {
		return nil, fmt.Errorf("open embedded handbook package: %w", err)
	}
	defer gzipReader.Close()

	var manifestRaw []byte
	fileCount := 0
	documentCount := 0
	totalSize := int64(0)
	seenPaths := make(map[string]struct{})
	tarReader := tar.NewReader(gzipReader)
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("read embedded handbook package: %w", err)
		}
		if header.Typeflag != tar.TypeReg && header.Typeflag != tar.TypeRegA {
			return nil, fmt.Errorf("embedded handbook package contains non-regular entry %s", header.Name)
		}
		cleanName := path.Clean(strings.TrimSpace(header.Name))
		if cleanName == "." || strings.HasPrefix(cleanName, "../") || !strings.HasPrefix(cleanName, handbookContentRoot+"/") {
			return nil, fmt.Errorf("embedded handbook package contains invalid path %s", header.Name)
		}
		if _, exists := seenPaths[cleanName]; exists {
			return nil, fmt.Errorf("embedded handbook package contains duplicate path %s", cleanName)
		}
		seenPaths[cleanName] = struct{}{}
		if header.Size < 0 {
			return nil, fmt.Errorf("embedded handbook package contains invalid size for %s", cleanName)
		}
		fileCount++
		totalSize += header.Size
		if fileCount > maxHandbookFiles || totalSize > maxHandbookUncompressedSize {
			return nil, fmt.Errorf("embedded handbook package exceeds extraction limits")
		}
		if cleanName == handbookContentRoot+"/manifest.json" {
			manifestRaw, err = io.ReadAll(io.LimitReader(tarReader, 1<<20))
			if err != nil {
				return nil, fmt.Errorf("read embedded handbook manifest: %w", err)
			}
		}
		if strings.HasPrefix(cleanName, handbookContentRoot+"/api-docs/") && strings.HasSuffix(cleanName, ".json") {
			documentCount++
		}
	}
	if _, err := io.Copy(io.Discard, gzipReader); err != nil {
		return nil, fmt.Errorf("verify embedded handbook package: %w", err)
	}
	if len(manifestRaw) == 0 {
		return nil, fmt.Errorf("embedded handbook package has no references/manifest.json")
	}
	var sourceManifest struct {
		SchemaVersion   string `json:"schemaVersion"`
		CatalogVersion  string `json:"catalogVersion"`
		TotalOperations int    `json:"totalOperations"`
	}
	if err := json.Unmarshal(manifestRaw, &sourceManifest); err != nil {
		return nil, fmt.Errorf("decode embedded handbook manifest: %w", err)
	}
	if sourceManifest.SchemaVersion == "" || sourceManifest.CatalogVersion == "" || sourceManifest.TotalOperations <= 0 {
		return nil, fmt.Errorf("embedded handbook manifest is incomplete")
	}
	if documentCount != sourceManifest.TotalOperations {
		return nil, fmt.Errorf("embedded handbook contains %d operation documents, expected %d", documentCount, sourceManifest.TotalOperations)
	}
	digest := sha256.Sum256(archive)
	archiveCopy := append([]byte(nil), archive...)
	totalChunks := (len(archiveCopy) + handbookChunkBytes - 1) / handbookChunkBytes
	return &handbookPackage{
		archive: archiveCopy,
		manifest: HandbookPackageManifest{
			SchemaVersion:   sourceManifest.SchemaVersion,
			CatalogVersion:  sourceManifest.CatalogVersion,
			HandbookDigest:  fmt.Sprintf("sha256:%x", digest),
			TotalOperations: sourceManifest.TotalOperations,
			ManifestPath:    "/api/v1/agent/handbook/manifest",
			ChunkPath:       "/api/v1/agent/handbook/chunks/{index}",
			ArchiveFormat:   handbookArchiveFormat,
			ArchiveBytes:    len(archiveCopy),
			ChunkBytes:      handbookChunkBytes,
			TotalChunks:     totalChunks,
			ContentRoot:     handbookContentRoot,
		},
	}, nil
}

func (p *handbookPackage) validatedManifest(routes HandbookRouteExport) (HandbookPackageManifest, error) {
	if p == nil {
		return HandbookPackageManifest{}, fmt.Errorf("handbook package is unavailable")
	}
	if p.manifest.SchemaVersion != routes.SchemaVersion ||
		p.manifest.CatalogVersion != routes.CatalogVersion ||
		p.manifest.TotalOperations != routes.TotalRoutes {
		return HandbookPackageManifest{}, fmt.Errorf(
			"handbook package does not match running API: package=%s/%s/%d runtime=%s/%s/%d",
			p.manifest.SchemaVersion,
			p.manifest.CatalogVersion,
			p.manifest.TotalOperations,
			routes.SchemaVersion,
			routes.CatalogVersion,
			routes.TotalRoutes,
		)
	}
	return p.manifest, nil
}

func (p *handbookPackage) chunk(routes HandbookRouteExport, index int) (HandbookPackageChunk, error) {
	manifest, err := p.validatedManifest(routes)
	if err != nil {
		return HandbookPackageChunk{}, err
	}
	if index < 0 || index >= manifest.TotalChunks {
		return HandbookPackageChunk{}, fmt.Errorf("handbook chunk index %d is outside 0..%d", index, manifest.TotalChunks-1)
	}
	start := index * manifest.ChunkBytes
	end := start + manifest.ChunkBytes
	if end > len(p.archive) {
		end = len(p.archive)
	}
	raw := p.archive[start:end]
	return HandbookPackageChunk{
		HandbookDigest: manifest.HandbookDigest,
		Index:          index,
		TotalChunks:    manifest.TotalChunks,
		Bytes:          len(raw),
		Encoding:       "base64",
		Data:           base64.StdEncoding.EncodeToString(raw),
	}, nil
}
