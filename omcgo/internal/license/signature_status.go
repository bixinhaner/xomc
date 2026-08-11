package license

// SignatureStatus 是旧项目 License 数字签名校验结果。
type SignatureStatus string

const (
	SignatureUnverified SignatureStatus = "unverified"
	SignatureVerified   SignatureStatus = "verified"
	SignatureInvalid    SignatureStatus = "invalid"
)
