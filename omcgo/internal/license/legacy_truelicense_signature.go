package license

import (
	"bytes"
	"crypto/dsa"
	"crypto/sha1"
	"crypto/x509"
	"encoding/asn1"
	"encoding/binary"
	"errors"
	"fmt"
	"math/big"
	"strings"
)

const (
	legacyJKSMagic          = 0xFEEDFEED
	legacyJKSVersion        = 2
	legacyJKSTrustedCertTag = 2
	legacyJKSIntegrityText  = "Mighty Aphrodite"
)

// LegacyJKSOptions controls compatibility handling for old reference
// keystores. Production should keep VerifyIntegrity true.
type LegacyJKSOptions struct {
	VerifyIntegrity bool
}

// VerifyLegacyTrueLicenseSignature verifies GenericCertificate.signature over
// the exact UTF-8 bytes of GenericCertificate.encoded using SHA1withDSA.
func VerifyLegacyTrueLicenseSignature(artifact *LegacyTrueLicenseArtifact, keyStoreBytes []byte, storePassword, alias string) error {
	return VerifyLegacyTrueLicenseSignatureWithOptions(artifact, keyStoreBytes, storePassword, alias, LegacyJKSOptions{VerifyIntegrity: true})
}

// VerifyLegacyTrueLicenseSignatureWithOptions verifies a legacy DSA signature
// and loads the matching trusted certificate from a JKS keystore.
func VerifyLegacyTrueLicenseSignatureWithOptions(artifact *LegacyTrueLicenseArtifact, keyStoreBytes []byte, storePassword, alias string, options LegacyJKSOptions) error {
	if artifact == nil {
		return errors.New("legacy license artifact is nil")
	}
	if artifact.SignatureAlgorithm != "SHA1withDSA" {
		return fmt.Errorf("unsupported legacy signature algorithm %q", artifact.SignatureAlgorithm)
	}
	if artifact.SignatureEncoding != "US-ASCII/Base64" {
		return fmt.Errorf("unsupported legacy signature encoding %q", artifact.SignatureEncoding)
	}
	publicKey, err := loadLegacyJKSDSAPublicKey(keyStoreBytes, storePassword, alias, options.VerifyIntegrity)
	if err != nil {
		return fmt.Errorf("load legacy JKS public key: %w", err)
	}
	signature, err := artifact.DecodeLegacySignature()
	if err != nil {
		return err
	}
	digest := sha1.Sum([]byte(artifact.EncodedContent))
	var dsaSignature struct {
		R *big.Int
		S *big.Int
	}
	if rest, err := asn1.Unmarshal(signature, &dsaSignature); err != nil || len(rest) != 0 || dsaSignature.R == nil || dsaSignature.S == nil || !dsa.Verify(publicKey, digest[:], dsaSignature.R, dsaSignature.S) {
		return errors.New("legacy license DSA signature verification failed")
	}
	return nil
}

func loadLegacyJKSDSAPublicKey(raw []byte, password, alias string, verifyIntegrity bool) (*dsa.PublicKey, error) {
	if len(raw) < 16 {
		return nil, errors.New("JKS is truncated")
	}
	if binary.BigEndian.Uint32(raw[:4]) != legacyJKSMagic {
		return nil, errors.New("invalid JKS magic")
	}
	version := binary.BigEndian.Uint32(raw[4:8])
	if version != 1 && version != legacyJKSVersion {
		return nil, fmt.Errorf("unsupported JKS version %d", version)
	}
	entryCount := binary.BigEndian.Uint32(raw[8:12])
	if entryCount > 10000 {
		return nil, fmt.Errorf("invalid JKS entry count %d", entryCount)
	}

	checksumOffset := len(raw) - sha1.Size
	if checksumOffset <= 12 {
		return nil, errors.New("JKS checksum is missing")
	}
	passwordBytes := javaUTF16Password(password)
	checksumInput := make([]byte, 0, checksumOffset+len(passwordBytes)+len(legacyJKSIntegrityText))
	checksumInput = append(checksumInput, passwordBytes...)
	checksumInput = append(checksumInput, legacyJKSIntegrityText...)
	checksumInput = append(checksumInput, raw[:checksumOffset]...)
	expected := sha1.Sum(checksumInput)
	if verifyIntegrity && !bytes.Equal(expected[:], raw[checksumOffset:]) {
		return nil, errors.New("JKS integrity check failed")
	}

	reader := bytes.NewReader(raw[12:checksumOffset])
	for i := uint32(0); i < entryCount; i++ {
		tag, err := readJKSUint32(reader)
		if err != nil {
			return nil, fmt.Errorf("read JKS entry %d tag: %w", i, err)
		}
		entryAlias, err := readJKSUTF(reader)
		if err != nil {
			return nil, fmt.Errorf("read JKS entry %d alias: %w", i, err)
		}
		if _, err := readJKSInt64(reader); err != nil {
			return nil, fmt.Errorf("read JKS entry %d timestamp: %w", i, err)
		}

		switch tag {
		case legacyJKSTrustedCertTag:
			certType, err := readJKSUTF(reader)
			if err != nil {
				return nil, fmt.Errorf("read JKS certificate type: %w", err)
			}
			certBytes, err := readJKSBytes(reader)
			if err != nil {
				return nil, fmt.Errorf("read JKS certificate: %w", err)
			}
			if alias != "" && !strings.EqualFold(alias, entryAlias) {
				continue
			}
			if certType != "X.509" {
				return nil, fmt.Errorf("unsupported JKS certificate type %q", certType)
			}
			certificate, err := x509.ParseCertificate(certBytes)
			if err != nil {
				return nil, fmt.Errorf("parse JKS X.509 certificate: %w", err)
			}
			publicKey, ok := certificate.PublicKey.(*dsa.PublicKey)
			if !ok {
				return nil, fmt.Errorf("JKS certificate public key is %T, want DSA", certificate.PublicKey)
			}
			return publicKey, nil
		case 1:
			if err := skipJKSPrivateKeyEntry(reader); err != nil {
				return nil, fmt.Errorf("skip JKS private entry: %w", err)
			}
		default:
			return nil, fmt.Errorf("unsupported JKS entry tag %d", tag)
		}
	}
	return nil, fmt.Errorf("JKS trusted certificate alias %q not found", alias)
}

func javaUTF16Password(password string) []byte {
	result := make([]byte, 0, len(password)*2)
	for _, runeValue := range []rune(password) {
		if runeValue <= 0xffff {
			result = append(result, byte(runeValue>>8), byte(runeValue))
			continue
		}
		runeValue -= 0x10000
		hi, lo := rune(0xd800)+(runeValue>>10), rune(0xdc00)+(runeValue&0x3ff)
		result = append(result, byte(hi>>8), byte(hi), byte(lo>>8), byte(lo))
	}
	return result
}

func readJKSUint32(reader *bytes.Reader) (uint32, error) {
	var value uint32
	if err := binary.Read(reader, binary.BigEndian, &value); err != nil {
		return 0, err
	}
	return value, nil
}

func readJKSInt64(reader *bytes.Reader) (int64, error) {
	var value int64
	if err := binary.Read(reader, binary.BigEndian, &value); err != nil {
		return 0, err
	}
	return value, nil
}

func readJKSUTF(reader *bytes.Reader) (string, error) {
	length, err := readJKSUint16(reader)
	if err != nil {
		return "", err
	}
	if int(length) > reader.Len() {
		return "", errors.New("JKS UTF string exceeds remaining bytes")
	}
	value := make([]byte, length)
	if _, err := reader.Read(value); err != nil {
		return "", err
	}
	return string(value), nil
}

func readJKSUint16(reader *bytes.Reader) (uint16, error) {
	var value uint16
	if err := binary.Read(reader, binary.BigEndian, &value); err != nil {
		return 0, err
	}
	return value, nil
}

func readJKSBytes(reader *bytes.Reader) ([]byte, error) {
	length, err := readJKSUint32(reader)
	if err != nil {
		return nil, err
	}
	if uint64(length) > uint64(reader.Len()) {
		return nil, errors.New("JKS byte field exceeds remaining bytes")
	}
	value := make([]byte, length)
	if _, err := reader.Read(value); err != nil {
		return nil, err
	}
	return value, nil
}

func skipJKSPrivateKeyEntry(reader *bytes.Reader) error {
	if _, err := readJKSBytes(reader); err != nil {
		return fmt.Errorf("read encrypted private key: %w", err)
	}
	count, err := readJKSUint32(reader)
	if err != nil {
		return err
	}
	for i := uint32(0); i < count; i++ {
		if _, err := readJKSUTF(reader); err != nil {
			return err
		}
		if _, err := readJKSBytes(reader); err != nil {
			return err
		}
	}
	return nil
}
