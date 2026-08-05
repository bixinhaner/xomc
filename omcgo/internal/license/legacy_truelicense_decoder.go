package license

import (
	"bytes"
	"compress/gzip"
	"crypto/cipher"
	"crypto/des"
	"crypto/md5"
	"crypto/sha256"
	"encoding/base64"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
)

const (
	legacyPBEAlgorithm  = "PBEWithMD5AndDES"
	legacyPBEIterations = 2005
)

var legacyPBESalt = [8]byte{0xce, 0xfb, 0xde, 0xac, 0x05, 0x02, 0x19, 0x71}

// LegacyTrueLicenseArtifact contains the original legacy license and its
// decoded certificate metadata. RawBytes is never modified during decoding.
type LegacyTrueLicenseArtifact struct {
	RawBytes           []byte
	SHA256             string
	DecodedXML         []byte
	EncodedContent     string
	Signature          string
	SignatureAlgorithm string
	SignatureEncoding  string
	StandardFields     map[string]LegacyXMLValue
	Extra              map[string]LegacyXMLValue
	UnknownFields      map[string]LegacyXMLValue
}

// LegacyXMLValue preserves the XML scalar type instead of coercing every value
// to a string too early.
type LegacyXMLValue struct {
	Kind  string
	Value string
}

// DecodeLegacyTrueLicense decodes the TrueLicense 1.x container used by the
// legacy OMC. It deliberately does not verify the DSA signature; callers must
// provide the trusted public key and call VerifyLegacyTrueLicenseSignature.
func DecodeLegacyTrueLicense(raw []byte, password string) (*LegacyTrueLicenseArtifact, error) {
	if len(raw) == 0 {
		return nil, errors.New("legacy license is empty")
	}
	if strings.TrimSpace(password) == "" {
		return nil, errors.New("legacy license cipher password is required")
	}

	decoded, err := decryptLegacyTrueLicense(raw, password)
	if err != nil {
		return nil, fmt.Errorf("legacy license decrypt: %w", err)
	}
	artifact := &LegacyTrueLicenseArtifact{
		RawBytes:       append([]byte(nil), raw...),
		SHA256:         hexSHA256(raw),
		DecodedXML:     decoded,
		StandardFields: make(map[string]LegacyXMLValue),
		Extra:          make(map[string]LegacyXMLValue),
		UnknownFields:  make(map[string]LegacyXMLValue),
	}
	if err := parseLegacyCertificate(decoded, artifact); err != nil {
		return nil, fmt.Errorf("legacy license XML: %w", err)
	}
	return artifact, nil
}

func decryptLegacyTrueLicense(raw []byte, password string) ([]byte, error) {
	digest := md5.Sum(append([]byte(password), legacyPBESalt[:]...))
	state := digest[:]
	for i := 1; i < legacyPBEIterations; i++ {
		digest = md5.Sum(state)
		state = digest[:]
	}

	block, err := des.NewCipher(state[:des.BlockSize])
	if err != nil {
		return nil, fmt.Errorf("create DES cipher: %w", err)
	}
	if len(raw)%des.BlockSize != 0 {
		return nil, fmt.Errorf("ciphertext length %d is not aligned to DES block size", len(raw))
	}
	decrypted := make([]byte, len(raw))
	iv := append([]byte(nil), state[des.BlockSize:2*des.BlockSize]...)
	cipher.NewCBCDecrypter(block, iv).CryptBlocks(decrypted, raw)

	padding := int(decrypted[len(decrypted)-1])
	if padding < 1 || padding > des.BlockSize || padding > len(decrypted) {
		return nil, errors.New("invalid PKCS#5 padding")
	}
	for _, b := range decrypted[len(decrypted)-padding:] {
		if int(b) != padding {
			return nil, errors.New("invalid PKCS#5 padding")
		}
	}
	decrypted = decrypted[:len(decrypted)-padding]

	reader, err := gzip.NewReader(bytes.NewReader(decrypted))
	if err != nil {
		return nil, fmt.Errorf("open gzip payload: %w", err)
	}
	payload, readErr := io.ReadAll(reader)
	closeErr := reader.Close()
	if readErr != nil {
		return nil, fmt.Errorf("read gzip payload: %w", readErr)
	}
	if closeErr != nil {
		return nil, fmt.Errorf("close gzip payload: %w", closeErr)
	}
	return payload, nil
}

func hexSHA256(raw []byte) string {
	sum := sha256.Sum256(raw)
	return fmt.Sprintf("%x", sum[:])
}

// certificateXML mirrors only the XML envelope emitted by XMLEncoder. The
// nested encoded XML is parsed separately because it is stored as escaped text.
type certificateXML struct {
	Object struct {
		Values []xmlVoid `xml:"void"`
	} `xml:"object"`
}

type xmlVoid struct {
	Property string      `xml:"property,attr"`
	Method   string      `xml:"method,attr"`
	String   *string     `xml:"string"`
	Object   []xmlObject `xml:"object"`
}

type xmlObject struct {
	Class  string  `xml:"class,attr"`
	Long   *int64  `xml:"long"`
	Int    *int    `xml:"int"`
	String *string `xml:"string"`
}

func parseLegacyCertificate(rawXML []byte, artifact *LegacyTrueLicenseArtifact) error {
	var outer certificateXML
	if err := xml.Unmarshal(rawXML, &outer); err != nil {
		return fmt.Errorf("decode certificate envelope: %w", err)
	}
	for _, value := range outer.Object.Values {
		if value.Property == "encoded" && value.String != nil {
			artifact.EncodedContent = *value.String
		}
		if value.Property == "signature" && value.String != nil {
			artifact.Signature = *value.String
		}
		if value.Property == "signatureAlgorithm" && value.String != nil {
			artifact.SignatureAlgorithm = *value.String
		}
		if value.Property == "signatureEncoding" && value.String != nil {
			artifact.SignatureEncoding = *value.String
		}
	}
	if artifact.EncodedContent == "" {
		return errors.New("certificate encoded content is missing")
	}
	return parseLegacyContent([]byte(artifact.EncodedContent), artifact)
}

func parseLegacyContent(rawXML []byte, artifact *LegacyTrueLicenseArtifact) error {
	var content legacyContentXML
	if err := xml.Unmarshal(rawXML, &content); err != nil {
		return fmt.Errorf("decode license content: %w", err)
	}
	for _, value := range content.Values {
		if value.Property == "extra" {
			for _, entry := range value.ExtraEntries {
				artifact.Extra[entry.Key] = entry.Value
			}
			continue
		}
		if value.Property != "" {
			artifact.StandardFields[value.Property] = value.Scalar
		}
	}
	return nil
}

type legacyContentXML struct {
	Values []legacyContentVoid `xml:"object>void"`
}

type legacyContentVoid struct {
	Property     string           `xml:"property,attr"`
	Scalar       LegacyXMLValue   `xml:"-"`
	String       *string          `xml:"string"`
	Int          *int             `xml:"int"`
	Long         *int64           `xml:"long"`
	Object       *legacyObjectXML `xml:"object"`
	ExtraEntries []legacyMapEntry `xml:"-"`
}

type legacyObjectXML struct {
	Class   string           `xml:"class,attr"`
	String  *string          `xml:"string"`
	Int     *int             `xml:"int"`
	Long    *int64           `xml:"long"`
	Entries []legacyMapEntry `xml:"void"`
}

type legacyMapEntry struct {
	Method string         `xml:"method,attr"`
	String []*string      `xml:"string"`
	Int    []*int         `xml:"int"`
	Long   []*int64       `xml:"long"`
	Key    string         `xml:"-"`
	Value  LegacyXMLValue `xml:"-"`
}

func (v *legacyContentVoid) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	type alias legacyContentVoid
	var a alias
	if err := d.DecodeElement(&a, &start); err != nil {
		return err
	}
	*v = legacyContentVoid(a)
	if v.Object != nil {
		v.ExtraEntries = v.Object.Entries
	}
	switch {
	case v.String != nil:
		v.Scalar = LegacyXMLValue{Kind: "string", Value: *v.String}
	case v.Int != nil:
		v.Scalar = LegacyXMLValue{Kind: "int", Value: strconv.Itoa(*v.Int)}
	case v.Long != nil:
		v.Scalar = LegacyXMLValue{Kind: "long", Value: strconv.FormatInt(*v.Long, 10)}
	case v.Object != nil && v.Object.Long != nil:
		v.Scalar = LegacyXMLValue{Kind: "long", Value: strconv.FormatInt(*v.Object.Long, 10)}
	case v.Object != nil && v.Object.Int != nil:
		v.Scalar = LegacyXMLValue{Kind: "int", Value: strconv.Itoa(*v.Object.Int)}
	case v.Object != nil && v.Object.String != nil:
		v.Scalar = LegacyXMLValue{Kind: "string", Value: *v.Object.String}
	default:
		v.Scalar = LegacyXMLValue{Kind: "object", Value: ""}
	}
	for i := range v.ExtraEntries {
		entry := &v.ExtraEntries[i]
		if entry.Method != "put" || len(entry.String) == 0 {
			continue
		}
		entry.Key = *entry.String[0]
		switch {
		case len(entry.String) > 1:
			entry.Value = LegacyXMLValue{Kind: "string", Value: *entry.String[1]}
		case len(entry.Int) > 0:
			entry.Value = LegacyXMLValue{Kind: "int", Value: strconv.Itoa(*entry.Int[0])}
		case len(entry.Long) > 0:
			entry.Value = LegacyXMLValue{Kind: "long", Value: strconv.FormatInt(*entry.Long[0], 10)}
		}
	}
	return nil
}

func (e *legacyMapEntry) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	type alias legacyMapEntry
	var a alias
	if err := d.DecodeElement(&a, &start); err != nil {
		return err
	}
	*e = legacyMapEntry(a)
	return nil
}

// DecodeLegacySignature decodes the Base64 signature emitted by GenericCertificate.
func (a *LegacyTrueLicenseArtifact) DecodeLegacySignature() ([]byte, error) {
	if a == nil || a.Signature == "" {
		return nil, errors.New("legacy license signature is missing")
	}
	return base64.StdEncoding.DecodeString(a.Signature)
}
