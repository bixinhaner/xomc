package storageprotection

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const (
	defaultDockerRootDir     = "/var/lib/docker"
	defaultDockerVolumeGroup = "omcgo"
	defaultOMCLogsPath       = "/opt/omc/run/logs"
	defaultOMCDataPath       = "/opt/omc/data"
)

var protectedDataPathKeys = []struct {
	envKey string
	volume string
}{
	{envKey: "POSTGRES_DATA_PATH", volume: "pgdata"},
	{envKey: "TSDB_DATA_PATH", volume: "tsdbdata"},
	{envKey: "REDIS_DATA_PATH", volume: "redisdata"},
	{envKey: "REDIS_PM_DATA_PATH", volume: "redispmdata"},
	{envKey: "NATS_DATA_PATH", volume: "natsdata"},
	{envKey: "MINIO_DATA_PATH", volume: "miniodata"},
}

type ProtectedPath struct {
	ID   string
	Path string
}

type ProtectedPathResolver interface {
	ProtectedPaths(ctx context.Context) ([]ProtectedPath, error)
}

type DockerVolumeResolver interface {
	ResolveNamedVolume(volume string) (string, bool)
}

type EnvProtectedPathResolver struct {
	lookup         func(string) (string, bool)
	volumeResolver DockerVolumeResolver
}

func NewEnvProtectedPathResolver() *EnvProtectedPathResolver {
	resolver := &EnvProtectedPathResolver{lookup: os.LookupEnv}
	resolver.volumeResolver = dockerRootVolumeResolver{lookup: resolver.lookup}
	return resolver
}

func NewEnvProtectedPathResolverWithVolumeResolver(lookup func(string) (string, bool), volumeResolver DockerVolumeResolver) *EnvProtectedPathResolver {
	if lookup == nil {
		lookup = os.LookupEnv
	}
	return &EnvProtectedPathResolver{lookup: lookup, volumeResolver: volumeResolver}
}

func (r *EnvProtectedPathResolver) ProtectedPaths(context.Context) ([]ProtectedPath, error) {
	if r == nil {
		r = NewEnvProtectedPathResolver()
	}
	lookup := r.lookup
	if lookup == nil {
		lookup = os.LookupEnv
	}
	volumeResolver := r.volumeResolver
	if volumeResolver == nil {
		volumeResolver = dockerRootVolumeResolver{lookup: lookup}
	}

	paths := make([]ProtectedPath, 0, len(protectedDataPathKeys)+3)
	for _, item := range protectedDataPathKeys {
		if raw, _ := lookup(item.envKey); filepath.IsAbs(raw) {
			paths = append(paths, ProtectedPath{ID: item.envKey, Path: cleanHostPath(raw)})
			continue
		}
		if volumePath, ok := volumeResolver.ResolveNamedVolume(item.volume); ok && filepath.IsAbs(volumePath) {
			paths = append(paths, ProtectedPath{ID: item.envKey, Path: cleanHostPath(volumePath)})
		}
	}
	paths = append(paths,
		ProtectedPath{ID: "omc-logs", Path: envAbsolutePath(lookup, "OMCGO_HOST_LOGS_PATH", defaultOMCLogsPath)},
		ProtectedPath{ID: "omc-data", Path: envAbsolutePath(lookup, "OMCGO_HOST_DATA_PATH", defaultOMCDataPath)},
		ProtectedPath{ID: "docker-root", Path: dockerRootDir(lookup)},
	)

	return dedupeProtectedPaths(paths), nil
}

type dockerRootVolumeResolver struct {
	lookup func(string) (string, bool)
}

func (r dockerRootVolumeResolver) ResolveNamedVolume(volume string) (string, bool) {
	volume = strings.TrimSpace(volume)
	if volume == "" {
		return "", false
	}
	root := dockerRootDir(r.lookup)
	group := dockerVolumeGroup(r.lookup)
	name := volume
	if group != "" {
		name = group + "_" + volume
	}
	return filepath.Join(root, "volumes", name, "_data"), true
}

func dockerRootDir(lookup func(string) (string, bool)) string {
	for _, key := range []string{"OMCGO_DOCKER_ROOT_DIR", "DOCKER_ROOT_DIR"} {
		if value, ok := lookup(key); ok && filepath.IsAbs(value) {
			return cleanHostPath(value)
		}
	}
	return defaultDockerRootDir
}

func dockerVolumeGroup(lookup func(string) (string, bool)) string {
	for _, key := range []string{"OMCGO_DOCKER_VOLUME_PREFIX", "COMPOSE_PROJECT_NAME"} {
		if value, ok := lookup(key); ok {
			return strings.TrimSpace(value)
		}
	}
	return defaultDockerVolumeGroup
}

func envAbsolutePath(lookup func(string) (string, bool), key, fallback string) string {
	if value, ok := lookup(key); ok && filepath.IsAbs(value) {
		return cleanHostPath(value)
	}
	return cleanHostPath(fallback)
}

func cleanHostPath(value string) string {
	cleaned := filepath.Clean(strings.TrimSpace(value))
	if cleaned == "." {
		return ""
	}
	return cleaned
}

func dedupeProtectedPaths(paths []ProtectedPath) []ProtectedPath {
	byPath := make(map[string]ProtectedPath, len(paths))
	for _, item := range paths {
		if item.Path == "" || !filepath.IsAbs(item.Path) {
			continue
		}
		if existing, ok := byPath[item.Path]; ok {
			existing.ID = existing.ID + "," + item.ID
			byPath[item.Path] = existing
			continue
		}
		byPath[item.Path] = item
	}
	keys := make([]string, 0, len(byPath))
	for pathValue := range byPath {
		keys = append(keys, pathValue)
	}
	sort.Strings(keys)
	result := make([]ProtectedPath, 0, len(keys))
	for _, pathValue := range keys {
		result = append(result, byPath[pathValue])
	}
	return result
}
