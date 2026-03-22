package stun

import (
	"encoding/binary"
	"errors"
	"fmt"
	"net"
)

// STUN message types (RFC 5389).
const (
	TypeBindingRequest       uint16 = 0x0001
	TypeBindingResponse      uint16 = 0x0101
	TypeBindingErrorResponse uint16 = 0x0111
)

// STUN attribute types.
const (
	AttrMappedAddress    uint16 = 0x0001
	AttrResponseAddress  uint16 = 0x0002
	AttrChangeRequest    uint16 = 0x0003
	AttrSourceAddress    uint16 = 0x0004
	AttrChangedAddress   uint16 = 0x0005
	AttrErrorCode        uint16 = 0x0009
	AttrUnknownAttrs     uint16 = 0x000A
	AttrXORMappedAddress uint16 = 0x0020
)

// STUN header size is 20 bytes: 2 (type) + 2 (length) + 4 (magic cookie) + 12 (transaction id).
const headerSize = 20

// magicCookie is the fixed value in STUN headers (RFC 5389).
const magicCookie uint32 = 0x2112A442

var (
	ErrTooShort      = errors.New("stun: message too short")
	ErrInvalidMagic  = errors.New("stun: invalid magic cookie")
	ErrTruncatedAttr = errors.New("stun: truncated attribute")
)

// Message represents a STUN message.
type Message struct {
	Type          uint16
	TransactionID [12]byte
	Attributes    []Attribute
}

// Attribute represents a STUN message attribute.
type Attribute struct {
	Type  uint16
	Value []byte
}

// IsBindingRequest returns true if the message is a Binding Request.
func (m *Message) IsBindingRequest() bool {
	return m.Type == TypeBindingRequest
}

// ResponseAddress returns the RESPONSE-ADDRESS attribute value if present.
func (m *Message) ResponseAddress() *net.UDPAddr {
	for _, attr := range m.Attributes {
		if attr.Type == AttrResponseAddress {
			return decodeAddressAttr(attr.Value)
		}
	}
	return nil
}

// Decode parses a STUN message from raw bytes.
// Returns nil, nil if the data does not look like a STUN message (for non-standard fallback).
func Decode(data []byte) (*Message, error) {
	if len(data) < headerSize {
		return nil, ErrTooShort
	}

	msgType := binary.BigEndian.Uint16(data[0:2])
	msgLen := binary.BigEndian.Uint16(data[2:4])
	cookie := binary.BigEndian.Uint32(data[4:8])

	// RFC 5389: first two bits must be 0
	if msgType&0xC000 != 0 {
		return nil, nil // not a STUN message
	}

	if cookie != magicCookie {
		return nil, nil // not a STUN message
	}

	if int(msgLen) > len(data)-headerSize {
		return nil, ErrTruncatedAttr
	}

	msg := &Message{Type: msgType}
	copy(msg.TransactionID[:], data[8:20])

	// Parse attributes
	offset := headerSize
	end := headerSize + int(msgLen)
	for offset+4 <= end {
		attrType := binary.BigEndian.Uint16(data[offset : offset+2])
		attrLen := binary.BigEndian.Uint16(data[offset+2 : offset+4])
		offset += 4

		if offset+int(attrLen) > end {
			return nil, ErrTruncatedAttr
		}

		value := make([]byte, attrLen)
		copy(value, data[offset:offset+int(attrLen)])
		msg.Attributes = append(msg.Attributes, Attribute{Type: attrType, Value: value})

		// Attributes are padded to 4-byte boundaries
		offset += int(attrLen)
		if pad := int(attrLen) % 4; pad != 0 {
			offset += 4 - pad
		}
	}

	return msg, nil
}

// NewBindingResponse creates a Binding Response for a given request,
// including the MappedAddress attribute for the client's observed address.
func NewBindingResponse(txID [12]byte, mappedAddr *net.UDPAddr) *Message {
	msg := &Message{
		Type:          TypeBindingResponse,
		TransactionID: txID,
	}
	msg.Attributes = append(msg.Attributes, Attribute{
		Type:  AttrMappedAddress,
		Value: encodeAddressAttr(mappedAddr),
	})
	return msg
}

// NewBindingErrorResponse creates a Binding Error Response.
func NewBindingErrorResponse(txID [12]byte, code int, reason string) *Message {
	msg := &Message{
		Type:          TypeBindingErrorResponse,
		TransactionID: txID,
	}
	msg.Attributes = append(msg.Attributes, Attribute{
		Type:  AttrErrorCode,
		Value: encodeErrorCode(code, reason),
	})
	return msg
}

// Encode serializes a STUN message to bytes.
func (m *Message) Encode() []byte {
	// Calculate attributes length
	var attrsLen int
	for _, attr := range m.Attributes {
		attrsLen += 4 + len(attr.Value)
		if pad := len(attr.Value) % 4; pad != 0 {
			attrsLen += 4 - pad
		}
	}

	buf := make([]byte, headerSize+attrsLen)
	binary.BigEndian.PutUint16(buf[0:2], m.Type)
	binary.BigEndian.PutUint16(buf[2:4], uint16(attrsLen))
	binary.BigEndian.PutUint32(buf[4:8], magicCookie)
	copy(buf[8:20], m.TransactionID[:])

	offset := headerSize
	for _, attr := range m.Attributes {
		binary.BigEndian.PutUint16(buf[offset:offset+2], attr.Type)
		binary.BigEndian.PutUint16(buf[offset+2:offset+4], uint16(len(attr.Value)))
		copy(buf[offset+4:], attr.Value)
		offset += 4 + len(attr.Value)
		if pad := len(attr.Value) % 4; pad != 0 {
			offset += 4 - pad // padding bytes are already zero
		}
	}

	return buf
}

// encodeAddressAttr encodes a MAPPED-ADDRESS attribute value (RFC 5389 Section 15.1).
// Format: 1 byte padding, 1 byte family (0x01=IPv4), 2 bytes port, 4 bytes IPv4.
func encodeAddressAttr(addr *net.UDPAddr) []byte {
	ip4 := addr.IP.To4()
	if ip4 == nil {
		ip4 = net.IPv4zero.To4()
	}
	buf := make([]byte, 8)
	buf[0] = 0x00 // reserved
	buf[1] = 0x01 // IPv4 family
	binary.BigEndian.PutUint16(buf[2:4], uint16(addr.Port))
	copy(buf[4:8], ip4)
	return buf
}

// decodeAddressAttr decodes a MAPPED-ADDRESS attribute value.
func decodeAddressAttr(data []byte) *net.UDPAddr {
	if len(data) < 8 {
		return nil
	}
	family := data[1]
	if family != 0x01 { // only IPv4
		return nil
	}
	port := binary.BigEndian.Uint16(data[2:4])
	ip := net.IPv4(data[4], data[5], data[6], data[7])
	return &net.UDPAddr{IP: ip, Port: int(port)}
}

// encodeErrorCode encodes an ERROR-CODE attribute (RFC 5389 Section 15.6).
func encodeErrorCode(code int, reason string) []byte {
	class := code / 100
	number := code % 100
	reasonBytes := []byte(reason)
	buf := make([]byte, 4+len(reasonBytes))
	buf[2] = byte(class)
	buf[3] = byte(number)
	copy(buf[4:], reasonBytes)
	return buf
}

// IsSTUN checks if a raw UDP packet looks like a STUN message
// by checking the first two bits are 0 and the magic cookie matches.
func IsSTUN(data []byte) bool {
	if len(data) < headerSize {
		return false
	}
	if data[0]&0xC0 != 0 {
		return false
	}
	cookie := binary.BigEndian.Uint32(data[4:8])
	return cookie == magicCookie
}

// TypeString returns a human-readable name for a STUN message type.
func TypeString(t uint16) string {
	switch t {
	case TypeBindingRequest:
		return "Binding Request"
	case TypeBindingResponse:
		return "Binding Response"
	case TypeBindingErrorResponse:
		return "Binding Error Response"
	default:
		return fmt.Sprintf("Unknown(0x%04x)", t)
	}
}
