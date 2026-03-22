package stun

import (
	"encoding/binary"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsSTUN(t *testing.T) {
	tests := []struct {
		name string
		data []byte
		want bool
	}{
		{
			name: "valid stun header",
			data: buildMinimalSTUN(TypeBindingRequest, [12]byte{1, 2, 3}),
			want: true,
		},
		{
			name: "too short",
			data: []byte{0x00, 0x01},
			want: false,
		},
		{
			name: "wrong magic cookie",
			data: func() []byte {
				d := buildMinimalSTUN(TypeBindingRequest, [12]byte{})
				binary.BigEndian.PutUint32(d[4:8], 0xDEADBEEF)
				return d
			}(),
			want: false,
		},
		{
			name: "first two bits set",
			data: func() []byte {
				d := buildMinimalSTUN(TypeBindingRequest, [12]byte{})
				d[0] |= 0x80 // set MSB
				return d
			}(),
			want: false,
		},
		{
			name: "non-standard enb message",
			data: []byte("ENB_SN_12345"),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, IsSTUN(tt.data))
		})
	}
}

func TestDecode_BindingRequest(t *testing.T) {
	txID := [12]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C}
	data := buildMinimalSTUN(TypeBindingRequest, txID)

	msg, err := Decode(data)
	require.NoError(t, err)
	require.NotNil(t, msg)
	assert.Equal(t, TypeBindingRequest, msg.Type)
	assert.Equal(t, txID, msg.TransactionID)
	assert.True(t, msg.IsBindingRequest())
	assert.Empty(t, msg.Attributes)
}

func TestDecode_WithAttributes(t *testing.T) {
	txID := [12]byte{0xAA, 0xBB}
	data := buildMinimalSTUN(TypeBindingRequest, txID)

	// Append a MAPPED-ADDRESS attribute: type=0x0001, len=8, value=8 bytes
	addr := &net.UDPAddr{IP: net.IPv4(192, 168, 1, 100), Port: 5000}
	attrValue := encodeAddressAttr(addr)
	attr := make([]byte, 4+len(attrValue))
	binary.BigEndian.PutUint16(attr[0:2], AttrMappedAddress)
	binary.BigEndian.PutUint16(attr[2:4], uint16(len(attrValue)))
	copy(attr[4:], attrValue)

	// Update message length
	data = append(data, attr...)
	binary.BigEndian.PutUint16(data[2:4], uint16(len(attr)))

	msg, err := Decode(data)
	require.NoError(t, err)
	require.NotNil(t, msg)
	assert.Len(t, msg.Attributes, 1)
	assert.Equal(t, AttrMappedAddress, msg.Attributes[0].Type)
}

func TestDecode_NotSTUN(t *testing.T) {
	// Non-STUN data: wrong magic cookie
	data := make([]byte, 20)
	binary.BigEndian.PutUint16(data[0:2], 0x0001)
	binary.BigEndian.PutUint32(data[4:8], 0x00000000) // wrong magic

	msg, err := Decode(data)
	assert.NoError(t, err)
	assert.Nil(t, msg) // not a STUN message, nil without error
}

func TestDecode_TooShort(t *testing.T) {
	_, err := Decode([]byte{0x00})
	assert.ErrorIs(t, err, ErrTooShort)
}

func TestDecode_TruncatedAttribute(t *testing.T) {
	data := buildMinimalSTUN(TypeBindingRequest, [12]byte{})
	// Set message length to 8, but only append 4 bytes of attr header
	binary.BigEndian.PutUint16(data[2:4], 8)
	attr := make([]byte, 4)
	binary.BigEndian.PutUint16(attr[0:2], AttrMappedAddress)
	binary.BigEndian.PutUint16(attr[2:4], 100) // claim 100 bytes, but nothing follows
	data = append(data, attr...)

	_, err := Decode(data)
	assert.ErrorIs(t, err, ErrTruncatedAttr)
}

func TestEncodeDecode_Roundtrip(t *testing.T) {
	txID := [12]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C}
	addr := &net.UDPAddr{IP: net.IPv4(10, 0, 0, 1), Port: 12345}

	original := NewBindingResponse(txID, addr)
	encoded := original.Encode()

	decoded, err := Decode(encoded)
	require.NoError(t, err)
	require.NotNil(t, decoded)

	assert.Equal(t, TypeBindingResponse, decoded.Type)
	assert.Equal(t, txID, decoded.TransactionID)
	require.Len(t, decoded.Attributes, 1)
	assert.Equal(t, AttrMappedAddress, decoded.Attributes[0].Type)

	// Verify the mapped address
	decodedAddr := decodeAddressAttr(decoded.Attributes[0].Value)
	require.NotNil(t, decodedAddr)
	assert.Equal(t, addr.IP.To4(), decodedAddr.IP.To4())
	assert.Equal(t, addr.Port, decodedAddr.Port)
}

func TestNewBindingResponse(t *testing.T) {
	txID := [12]byte{0xFF}
	addr := &net.UDPAddr{IP: net.IPv4(203, 0, 113, 50), Port: 8080}

	resp := NewBindingResponse(txID, addr)
	assert.Equal(t, TypeBindingResponse, resp.Type)
	assert.Equal(t, txID, resp.TransactionID)
	require.Len(t, resp.Attributes, 1)
	assert.Equal(t, AttrMappedAddress, resp.Attributes[0].Type)
}

func TestNewBindingErrorResponse(t *testing.T) {
	txID := [12]byte{0xAA}
	resp := NewBindingErrorResponse(txID, 400, "Bad Request")

	assert.Equal(t, TypeBindingErrorResponse, resp.Type)
	assert.Equal(t, txID, resp.TransactionID)
	require.Len(t, resp.Attributes, 1)
	assert.Equal(t, AttrErrorCode, resp.Attributes[0].Type)

	// Verify error code encoding
	val := resp.Attributes[0].Value
	assert.Equal(t, byte(4), val[2])  // class = 4
	assert.Equal(t, byte(0), val[3])  // number = 0
	assert.Equal(t, "Bad Request", string(val[4:]))
}

func TestEncodeDecodeAddressAttr(t *testing.T) {
	tests := []struct {
		name string
		addr *net.UDPAddr
	}{
		{
			name: "standard address",
			addr: &net.UDPAddr{IP: net.IPv4(192, 168, 1, 1), Port: 3478},
		},
		{
			name: "high port",
			addr: &net.UDPAddr{IP: net.IPv4(10, 0, 0, 1), Port: 65535},
		},
		{
			name: "zero port",
			addr: &net.UDPAddr{IP: net.IPv4(1, 2, 3, 4), Port: 0},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encoded := encodeAddressAttr(tt.addr)
			decoded := decodeAddressAttr(encoded)
			require.NotNil(t, decoded)
			assert.Equal(t, tt.addr.IP.To4(), decoded.IP.To4())
			assert.Equal(t, tt.addr.Port, decoded.Port)
		})
	}
}

func TestDecodeAddressAttr_Invalid(t *testing.T) {
	// Too short
	assert.Nil(t, decodeAddressAttr([]byte{0x00}))

	// IPv6 family (not supported)
	data := make([]byte, 8)
	data[1] = 0x02 // IPv6
	assert.Nil(t, decodeAddressAttr(data))
}

func TestResponseAddress(t *testing.T) {
	msg := &Message{
		Type: TypeBindingRequest,
		Attributes: []Attribute{
			{Type: AttrChangeRequest, Value: []byte{0, 0, 0, 0}},
		},
	}

	// No RESPONSE-ADDRESS
	assert.Nil(t, msg.ResponseAddress())

	// Add RESPONSE-ADDRESS
	addr := &net.UDPAddr{IP: net.IPv4(10, 20, 30, 40), Port: 5060}
	msg.Attributes = append(msg.Attributes, Attribute{
		Type:  AttrResponseAddress,
		Value: encodeAddressAttr(addr),
	})

	ra := msg.ResponseAddress()
	require.NotNil(t, ra)
	assert.Equal(t, addr.IP.To4(), ra.IP.To4())
	assert.Equal(t, addr.Port, ra.Port)
}

func TestTypeString(t *testing.T) {
	assert.Equal(t, "Binding Request", TypeString(TypeBindingRequest))
	assert.Equal(t, "Binding Response", TypeString(TypeBindingResponse))
	assert.Equal(t, "Binding Error Response", TypeString(TypeBindingErrorResponse))
	assert.Contains(t, TypeString(0x9999), "Unknown")
}

// buildMinimalSTUN creates a minimal STUN message with no attributes.
func buildMinimalSTUN(msgType uint16, txID [12]byte) []byte {
	buf := make([]byte, headerSize)
	binary.BigEndian.PutUint16(buf[0:2], msgType)
	binary.BigEndian.PutUint16(buf[2:4], 0) // no attributes
	binary.BigEndian.PutUint32(buf[4:8], magicCookie)
	copy(buf[8:20], txID[:])
	return buf
}
