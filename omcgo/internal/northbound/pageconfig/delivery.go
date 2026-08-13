package pageconfig

import (
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"net"
	"net/textproto"
	"os"
	pathpkg "path"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

func (s *Service) deliverRunToTargets(ctx context.Context, run FileRun) error {
	if run.Status != RunStatusSuccess || strings.TrimSpace(run.ArtifactContent) == "" {
		return nil
	}
	targets, err := s.deliveryTargetsForRun(ctx, run)
	if err != nil {
		return err
	}
	if len(targets) == 0 {
		return nil
	}

	artifact, err := materializeRunArtifact(run)
	if err != nil {
		return err
	}
	for _, target := range targets {
		result := uploadRunArtifact(ctx, target, run, artifact)
		if _, err := s.repo.CreateEvent(ctx, eventFromDeliveryRun(target, result)); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) deliveryTargetsForRun(ctx context.Context, run FileRun) ([]DeliveryTarget, error) {
	scope := DeliveryScopeFile
	if run.ProfileKind == ProfileKindInventory {
		scope = DeliveryScopeInventory
	}
	return s.repo.ListActiveDeliveryTargets(ctx, scope, run.ProfileCode)
}

func uploadRunArtifact(ctx context.Context, target DeliveryTarget, run FileRun, artifact []byte) DeliveryUploadResult {
	start := time.Now()
	result := DeliveryUploadResult{
		Success:     false,
		Protocol:    target.Protocol,
		Host:        strings.TrimSpace(target.Host),
		Port:        target.Port,
		RemotePath:  remoteArtifactPath(target, run),
		Bytes:       int64(len(artifact)),
		RunID:       run.ID,
		ProfileKind: run.ProfileKind,
		ProfileCode: run.ProfileCode,
	}
	if result.Host == "" {
		result.Error = "delivery target host is empty"
		result.Message = result.Error
		return result
	}
	if result.Port == 0 {
		if target.Protocol == DeliveryProtocolSFTP {
			result.Port = 22
		} else {
			result.Port = 21
		}
		target.Port = result.Port
	}

	maxAttempts := target.RetryTimes + 1
	if maxAttempts < 1 {
		maxAttempts = 1
	}
	if maxAttempts > 21 {
		maxAttempts = 21
	}

	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		result.Attempts = attempt
		switch target.Protocol {
		case DeliveryProtocolSFTP:
			lastErr = uploadSFTP(ctx, target, result.RemotePath, artifact)
		default:
			lastErr = uploadFTP(ctx, target, result.RemotePath, artifact)
		}
		if lastErr == nil {
			result.Success = true
			result.LatencyMs = time.Since(start).Milliseconds()
			result.Message = fmt.Sprintf("uploaded %d bytes to %s", len(artifact), result.RemotePath)
			return result
		}
		if attempt < maxAttempts {
			select {
			case <-ctx.Done():
				lastErr = ctx.Err()
				attempt = maxAttempts
			case <-time.After(time.Duration(attempt) * 200 * time.Millisecond):
			}
		}
	}
	result.LatencyMs = time.Since(start).Milliseconds()
	result.Error = scrubSecret(firstNonEmpty(errorString(lastErr), "delivery upload failed"), target.Credential)
	result.Message = result.Error
	return result
}

func materializeRunArtifact(run FileRun) ([]byte, error) {
	content := []byte(run.ArtifactContent)
	if !run.CompressionEnabled {
		return content, nil
	}
	switch compressionFormatOrDefault(run.CompressionFormat) {
	case CompressionGz:
		var out bytes.Buffer
		gw := gzip.NewWriter(&out)
		if _, err := gw.Write(content); err != nil {
			_ = gw.Close()
			return nil, fmt.Errorf("gzip northbound artifact: %w", err)
		}
		if err := gw.Close(); err != nil {
			return nil, fmt.Errorf("close gzip northbound artifact: %w", err)
		}
		return out.Bytes(), nil
	default:
		var out bytes.Buffer
		zw := zip.NewWriter(&out)
		name := strings.TrimSuffix(run.ArtifactName, ".zip")
		name = strings.TrimSuffix(name, ".gz")
		if strings.TrimSpace(name) == "" {
			name = "northbound-artifact.txt"
		}
		w, err := zw.Create(pathpkg.Base(name))
		if err != nil {
			_ = zw.Close()
			return nil, fmt.Errorf("create zip northbound artifact: %w", err)
		}
		if _, err := w.Write(content); err != nil {
			_ = zw.Close()
			return nil, fmt.Errorf("write zip northbound artifact: %w", err)
		}
		if err := zw.Close(); err != nil {
			return nil, fmt.Errorf("close zip northbound artifact: %w", err)
		}
		return out.Bytes(), nil
	}
}

func uploadFTP(ctx context.Context, target DeliveryTarget, remotePath string, artifact []byte) error {
	if !target.PassiveMode {
		return fmt.Errorf("FTP active mode upload is not supported; enable passive_mode for target %s", target.Key)
	}
	timeout := deliveryTimeout(target)
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	conn, err := dialDeliveryTCP(ctx, target.Host, target.Port, timeout)
	if err != nil {
		return err
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(timeout))

	tc := textproto.NewConn(conn)
	defer tc.Close()
	if _, _, err := tc.ReadResponse(2); err != nil {
		return fmt.Errorf("FTP banner read failed: %w", err)
	}
	if err := ftpLogin(tc, target); err != nil {
		return err
	}
	if err := ftpSimpleCommand(tc, 2, "TYPE I"); err != nil {
		return err
	}
	if err := ftpMkdirAll(tc, pathpkg.Dir(remotePath)); err != nil {
		return err
	}
	dataConn, err := ftpOpenPassiveDataConn(ctx, tc, target, timeout)
	if err != nil {
		return err
	}
	defer dataConn.Close()
	if _, err := tc.Cmd("STOR %s", remotePath); err != nil {
		return fmt.Errorf("FTP STOR write failed: %w", err)
	}
	if _, _, err := tc.ReadResponse(1); err != nil {
		return fmt.Errorf("FTP STOR start failed: %w", err)
	}
	if _, err := dataConn.Write(artifact); err != nil {
		return fmt.Errorf("FTP data write failed: %w", err)
	}
	if err := dataConn.Close(); err != nil {
		return fmt.Errorf("FTP data close failed: %w", err)
	}
	if _, _, err := tc.ReadResponse(2); err != nil {
		return fmt.Errorf("FTP STOR finish failed: %w", err)
	}
	_ = ftpSimpleCommand(tc, 2, "QUIT")
	return nil
}

func uploadSFTP(ctx context.Context, target DeliveryTarget, remotePath string, artifact []byte) error {
	timeout := deliveryTimeout(target)
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cfg, err := sshClientConfig(target, timeout)
	if err != nil {
		return err
	}
	addr := net.JoinHostPort(strings.TrimSpace(target.Host), strconv.Itoa(target.Port))
	conn, err := dialDeliveryTCP(ctx, target.Host, target.Port, timeout)
	if err != nil {
		return err
	}
	if deadline, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(deadline)
	}
	sshConn, chans, reqs, err := ssh.NewClientConn(conn, addr, cfg)
	if err != nil {
		_ = conn.Close()
		return fmt.Errorf("SFTP SSH connection failed: %w", err)
	}
	sshClient := ssh.NewClient(sshConn, chans, reqs)
	defer sshClient.Close()

	client, err := sftp.NewClient(sshClient)
	if err != nil {
		return fmt.Errorf("SFTP client create failed: %w", err)
	}
	defer client.Close()

	if err := client.MkdirAll(pathpkg.Dir(remotePath)); err != nil {
		return fmt.Errorf("SFTP mkdir failed: %w", err)
	}
	file, err := client.OpenFile(remotePath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY)
	if err != nil {
		return fmt.Errorf("SFTP open remote file failed: %w", err)
	}
	if _, err := io.Copy(file, bytes.NewReader(artifact)); err != nil {
		_ = file.Close()
		return fmt.Errorf("SFTP write remote file failed: %w", err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("SFTP close remote file failed: %w", err)
	}
	return nil
}

func ftpLogin(tc *textproto.Conn, target DeliveryTarget) error {
	if _, err := tc.Cmd("USER %s", target.Username); err != nil {
		return fmt.Errorf("FTP USER write failed: %w", err)
	}
	code, _, err := tc.ReadResponse(0)
	if err != nil {
		return fmt.Errorf("FTP USER response read failed: %w", err)
	}
	if code == 230 {
		return nil
	}
	if code != 331 {
		return fmt.Errorf("FTP USER rejected: code %d", code)
	}
	if _, err := tc.Cmd("PASS %s", target.Credential); err != nil {
		return fmt.Errorf("FTP PASS write failed: %w", err)
	}
	code, msg, err := tc.ReadResponse(0)
	if err != nil {
		return fmt.Errorf("FTP PASS response read failed: %w", err)
	}
	if code < 200 || code >= 300 {
		return fmt.Errorf("FTP auth rejected: code %d %s", code, msg)
	}
	return nil
}

func ftpMkdirAll(tc *textproto.Conn, dir string) error {
	dir = cleanRemoteDir(dir)
	if dir == "/" {
		return nil
	}
	current := ""
	for _, part := range strings.Split(strings.Trim(dir, "/"), "/") {
		if part == "" {
			continue
		}
		current += "/" + part
		if _, err := tc.Cmd("MKD %s", current); err != nil {
			return fmt.Errorf("FTP MKD write failed: %w", err)
		}
		_, _, _ = tc.ReadResponse(0)
	}
	return nil
}

func ftpOpenPassiveDataConn(ctx context.Context, tc *textproto.Conn, target DeliveryTarget, timeout time.Duration) (net.Conn, error) {
	if _, err := tc.Cmd("PASV"); err != nil {
		return nil, fmt.Errorf("FTP PASV write failed: %w", err)
	}
	_, msg, err := tc.ReadResponse(2)
	if err != nil {
		return nil, fmt.Errorf("FTP PASV response failed: %w", err)
	}
	host, port, err := parsePASVAddress(msg)
	if err != nil {
		return nil, err
	}
	if host == "" || host == "0.0.0.0" {
		host = strings.TrimSpace(target.Host)
	}
	return dialDeliveryTCP(ctx, host, port, timeout)
}

func parsePASVAddress(message string) (string, int, error) {
	start := strings.Index(message, "(")
	end := strings.Index(message, ")")
	if start < 0 || end <= start {
		return "", 0, fmt.Errorf("FTP PASV response missing address: %s", message)
	}
	parts := strings.Split(message[start+1:end], ",")
	if len(parts) != 6 {
		return "", 0, fmt.Errorf("FTP PASV response has invalid address: %s", message)
	}
	values := make([]int, 6)
	for i, part := range parts {
		value, err := strconv.Atoi(strings.TrimSpace(part))
		if err != nil || value < 0 || value > 255 {
			return "", 0, fmt.Errorf("FTP PASV response has invalid byte: %s", message)
		}
		values[i] = value
	}
	host := fmt.Sprintf("%d.%d.%d.%d", values[0], values[1], values[2], values[3])
	port := values[4]*256 + values[5]
	return host, port, nil
}

func ftpSimpleCommand(tc *textproto.Conn, expect int, format string, args ...any) error {
	if _, err := tc.Cmd(format, args...); err != nil {
		return fmt.Errorf("FTP command write failed: %w", err)
	}
	if _, _, err := tc.ReadResponse(expect); err != nil {
		return fmt.Errorf("FTP command response failed: %w", err)
	}
	return nil
}

func sshClientConfig(target DeliveryTarget, timeout time.Duration) (*ssh.ClientConfig, error) {
	// Username + password only. SFTP host-key verification is intentionally disabled
	// (trusted internal OSS networks); private-key auth is not supported.
	return &ssh.ClientConfig{
		User:            target.Username,
		Auth:            []ssh.AuthMethod{ssh.Password(target.Credential)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         timeout,
	}, nil
}

func dialDeliveryTCP(ctx context.Context, host string, port int, timeout time.Duration) (net.Conn, error) {
	addr := net.JoinHostPort(strings.TrimSpace(host), strconv.Itoa(port))
	dialer := &net.Dialer{Timeout: timeout}
	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("dial %s failed: %w", addr, err)
	}
	return conn, nil
}

func deliveryTimeout(target DeliveryTarget) time.Duration {
	timeout := time.Duration(target.TimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	if timeout > 5*time.Minute {
		timeout = 5 * time.Minute
	}
	return timeout
}

func remoteArtifactPath(target DeliveryTarget, run FileRun) string {
	root := cleanRemoteDir(target.RemoteRoot)
	artifactDir := cleanRemoteDir(run.ArtifactPath)
	if artifactDir == "/" {
		return pathpkg.Join(root, safeRemoteBase(run.ArtifactName))
	}
	if strings.HasPrefix(artifactDir, root+"/") || artifactDir == root {
		return pathpkg.Join(artifactDir, safeRemoteBase(run.ArtifactName))
	}
	return pathpkg.Join(root, strings.TrimPrefix(artifactDir, "/"), safeRemoteBase(run.ArtifactName))
}

func cleanRemoteDir(dir string) string {
	dir = strings.TrimSpace(strings.ReplaceAll(dir, "\\", "/"))
	if dir == "" {
		return "/"
	}
	cleaned := pathpkg.Clean("/" + strings.TrimPrefix(dir, "/"))
	if cleaned == "." {
		return "/"
	}
	return cleaned
}

func safeRemoteBase(name string) string {
	name = strings.TrimSpace(strings.ReplaceAll(name, "\\", "/"))
	name = pathpkg.Base(name)
	if name == "." || name == "/" || strings.Contains(name, "\x00") {
		return "northbound-artifact.txt"
	}
	return name
}

func errorString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
