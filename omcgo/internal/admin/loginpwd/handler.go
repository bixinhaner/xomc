package loginpwd

import (
	"github.com/gin-gonic/gin"

	"github.com/omcgo/omcgo/internal/core/response"
)

// PublicKeyResponse 是 GET /auth/public-key 的响应体。
//
// 前端取到后用 Web Crypto API 把 publicKey（PEM 编码 SubjectPublicKeyInfo）
// 导入为 CryptoKey，再用 RSA-OAEP/SHA-256 加密 JSON 载荷 {password, ts, nonce}。
type PublicKeyResponse struct {
	KeyID     string `json:"key_id"`
	PublicKey string `json:"public_key"`
	Algorithm string `json:"algorithm"`
	Hash      string `json:"hash"`
}

// PublicKeyHandler 暴露公钥下发接口。
type PublicKeyHandler struct {
	cipher *Cipher
}

// NewPublicKeyHandler 构造 handler。
func NewPublicKeyHandler(cipher *Cipher) *PublicKeyHandler {
	return &PublicKeyHandler{cipher: cipher}
}

// Get handles GET /api/v1/auth/public-key.
func (h *PublicKeyHandler) Get(c *gin.Context) {
	response.OK(c, PublicKeyResponse{
		KeyID:     h.cipher.ActiveKeyID(),
		PublicKey: h.cipher.ActivePublicKeyPEM(),
		Algorithm: "RSA-OAEP",
		Hash:      "SHA-256",
	})
}
