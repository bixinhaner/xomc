package admin

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math/big"
	"time"

	"github.com/omcgo/omcgo/internal/core/components/redisx"
	"github.com/redis/go-redis/v9"
)

// captchaTTL caps how long a generated challenge remains valid for
// verification. 5 minutes balances UX (operator may pause to read /
// retype) against limiting an attacker's window to brute-force the
// 32-symbol × 5-char keyspace (~33M combinations per challenge).
const captchaTTL = 5 * time.Minute

// captchaCharset is the alphabet drawn into the image. T-0031 / R-203
// excludes confusable glyphs (0/O, 1/I/L) so end users don't fail
// verification on visual ambiguity. Result: 31 characters → 31^5 ≈ 28.6M
// possible answers, comfortably > the brute-force budget allowed by
// captchaTTL × per-IP rate limit.
const captchaCharset = "23456789ABCDEFGHJKMNPQRSTUVWXYZ"

// captchaLength is the answer length. 5 keeps the image small enough
// for the login form (120×40 px) while preserving keyspace.
const captchaLength = 5

const (
	captchaImageWidth  = 120
	captchaImageHeight = 40
)

// CaptchaService generates and validates image-based CAPTCHA challenges
// using Redis for one-time-use answer storage. T-0031 / R-203: replaced
// the math-text challenge ("12+34=?") with a noisy bitmap render so OCR
// and trivial LLM solvers no longer succeed in zero-shot.
type CaptchaService struct {
	redis redis.UniversalClient
}

// NewCaptchaService creates a new CaptchaService.
func NewCaptchaService(client redis.UniversalClient) *CaptchaService {
	return &CaptchaService{redis: client}
}

// CaptchaChallenge is the response containing a CAPTCHA challenge for the
// client. T-0031: the legacy `Question` field is replaced by `Image`,
// which carries a base64-encoded PNG data URL the FE renders directly via
// `<img src={image}/>`. No font / image asset shipping needed.
type CaptchaChallenge struct {
	CaptchaID string `json:"captcha_id"`
	Image     string `json:"image"` // data:image/png;base64,...
}

// Generate creates a new image CAPTCHA challenge and stores the answer
// in Redis. The answer is stored UPPERCASE so Verify can canonicalize
// case-insensitively without exposing user-submitted casing patterns to
// timing-side-channel observers.
func (s *CaptchaService) Generate(ctx context.Context) (*CaptchaChallenge, error) {
	id, err := generateCaptchaID()
	if err != nil {
		return nil, fmt.Errorf("generate captcha id: %w", err)
	}

	answer, err := randomCaptchaAnswer()
	if err != nil {
		return nil, fmt.Errorf("generate captcha answer: %w", err)
	}

	imgData, err := renderCaptchaImage(answer)
	if err != nil {
		return nil, fmt.Errorf("render captcha image: %w", err)
	}

	key := redisx.Keys.AuthCaptcha(id)
	if err := s.redis.Set(ctx, key, answer, captchaTTL).Err(); err != nil {
		return nil, fmt.Errorf("store captcha answer: %w", err)
	}

	return &CaptchaChallenge{
		CaptchaID: id,
		Image:     "data:image/png;base64," + base64.StdEncoding.EncodeToString(imgData),
	}, nil
}

// Verify checks the provided answer against the stored answer and
// deletes it (one-time use). Comparison is case-insensitive — operators
// can type lowercase and still succeed. Returns false on any error
// (Redis miss, expired key, mismatch) without distinguishing the
// failure mode in the bool return — callers should not leak which
// branch was taken to the user.
func (s *CaptchaService) Verify(ctx context.Context, captchaID, answer string) bool {
	if captchaID == "" || answer == "" {
		return false
	}

	key := redisx.Keys.AuthCaptcha(captchaID)
	stored, err := s.redis.GetDel(ctx, key).Result()
	if err != nil {
		return false
	}

	return stored == toUpperASCII(answer)
}

// generateCaptchaID returns a 32-hex-char (16 random bytes) identifier
// used as the Redis lookup key. Cryptographically random; collision
// probability negligible.
func generateCaptchaID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// randomCaptchaAnswer picks captchaLength characters from captchaCharset
// using crypto/rand (NOT math/rand — the answer is the security-relevant
// secret for the challenge window).
func randomCaptchaAnswer() (string, error) {
	out := make([]byte, captchaLength)
	for i := range out {
		idx, err := rand.Int(rand.Reader, big.NewInt(int64(len(captchaCharset))))
		if err != nil {
			return "", err
		}
		out[i] = captchaCharset[idx.Int64()]
	}
	return string(out), nil
}

// toUpperASCII upper-cases an ASCII string without locale-aware unicode
// behaviour. Used so case-insensitive Verify doesn't pull in `strings.ToUpper`
// for what is by construction an ASCII-only domain.
func toUpperASCII(s string) string {
	out := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'a' && c <= 'z' {
			c -= 'a' - 'A'
		}
		out[i] = c
	}
	return string(out)
}

// renderCaptchaImage produces a PNG-encoded byte slice depicting the
// supplied answer string with random noise. Uses a hand-rolled 5×7
// bitmap font for the captcha character set so no third-party font /
// image dependency is needed (T-0031 §2 keep deps minimal).
func renderCaptchaImage(answer string) ([]byte, error) {
	img := image.NewRGBA(image.Rect(0, 0, captchaImageWidth, captchaImageHeight))
	bg := color.RGBA{R: 240, G: 240, B: 240, A: 255}
	draw.Draw(img, img.Bounds(), &image.Uniform{C: bg}, image.Point{}, draw.Src)

	// Draw glyphs spaced evenly across the canvas. Each glyph cell is
	// 5×7 pixels; we scale ×3 for ~15×21 visible size and stagger Y
	// position by a small jitter to defeat trivial OCR baseline detection.
	const cellWidth = 18
	const xMargin = (captchaImageWidth - cellWidth*captchaLength) / 2
	for i, ch := range answer {
		jitter := mustRandInt(5) - 2 // -2..+2 px vertical jitter
		drawGlyph(img, byte(ch), xMargin+i*cellWidth, 8+jitter, glyphColorFor(i))
	}

	// Noise overlay: ~50 random dots + 4 random lines. Tuned to be
	// visible enough to confuse OCR but sparse enough to keep the
	// answer readable to humans. Uses crypto/rand for noise too so the
	// pattern can't be predicted from a generation timestamp.
	for i := 0; i < 50; i++ {
		x := mustRandInt(captchaImageWidth)
		y := mustRandInt(captchaImageHeight)
		img.Set(x, y, color.RGBA{R: 180, G: 180, B: 180, A: 255})
	}
	for i := 0; i < 4; i++ {
		x1 := mustRandInt(captchaImageWidth)
		y1 := mustRandInt(captchaImageHeight)
		x2 := mustRandInt(captchaImageWidth)
		y2 := mustRandInt(captchaImageHeight)
		drawLine(img, x1, y1, x2, y2, color.RGBA{R: 150, G: 150, B: 150, A: 255})
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, fmt.Errorf("png encode: %w", err)
	}
	return buf.Bytes(), nil
}

// glyphColorFor cycles through a palette of dark colors per glyph index
// so the answer text doesn't look monochromatic (improves anti-OCR).
func glyphColorFor(i int) color.RGBA {
	palette := []color.RGBA{
		{R: 30, G: 30, B: 120, A: 255}, // navy
		{R: 120, G: 30, B: 30, A: 255}, // dark red
		{R: 30, G: 100, B: 30, A: 255}, // dark green
		{R: 80, G: 30, B: 80, A: 255},  // purple
		{R: 60, G: 60, B: 60, A: 255},  // dark grey
	}
	return palette[i%len(palette)]
}

// mustRandInt returns a uniformly-random int in [0, n) using crypto/rand.
// Panics on error — only called from CAPTCHA generation paths where
// rand.Reader failure is a fatal-class condition the caller can't
// reasonably recover from inside a render loop.
func mustRandInt(n int) int {
	v, err := rand.Int(rand.Reader, big.NewInt(int64(n)))
	if err != nil {
		// rand.Reader on Linux is /dev/urandom; failure is genuinely
		// fatal. Surfacing as panic stops a corrupted CAPTCHA from
		// being served, which is the safe default for an auth gate.
		panic(fmt.Sprintf("captcha rand failure: %v", err))
	}
	return int(v.Int64())
}

// drawLine draws a 1-pixel-wide line between two points using a
// minimal Bresenham implementation — stdlib has no line primitive in
// image/draw.
func drawLine(img *image.RGBA, x1, y1, x2, y2 int, c color.RGBA) {
	dx := abs(x2 - x1)
	dy := abs(y2 - y1)
	sx := -1
	if x1 < x2 {
		sx = 1
	}
	sy := -1
	if y1 < y2 {
		sy = 1
	}
	err := dx - dy
	for {
		img.Set(x1, y1, c)
		if x1 == x2 && y1 == y2 {
			break
		}
		e2 := 2 * err
		if e2 > -dy {
			err -= dy
			x1 += sx
		}
		if e2 < dx {
			err += dx
			y1 += sy
		}
	}
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// drawGlyph renders a single character as a 5×7 bitmap scaled ×3 at
// (x,y). Each row in glyph data is 5 bits LSB-first; only the
// captchaCharset characters are guaranteed to have glyphs — others
// render as a solid block.
func drawGlyph(img *image.RGBA, ch byte, x, y int, c color.RGBA) {
	bitmap, ok := glyphBitmaps[ch]
	if !ok {
		// Unknown char: solid filled block (defensive).
		for px := 0; px < 5; px++ {
			for py := 0; py < 7; py++ {
				fillScaled(img, x+px*3, y+py*3, 3, c)
			}
		}
		return
	}
	for py, row := range bitmap {
		for px := 0; px < 5; px++ {
			if row&(1<<(4-px)) != 0 {
				fillScaled(img, x+px*3, y+py*3, 3, c)
			}
		}
	}
}

// fillScaled paints a size×size block at (x, y).
func fillScaled(img *image.RGBA, x, y, size int, c color.RGBA) {
	for dy := 0; dy < size; dy++ {
		for dx := 0; dx < size; dx++ {
			img.Set(x+dx, y+dy, c)
		}
	}
}

// glyphBitmaps maps each captchaCharset character to a 5-wide × 7-tall
// pixel mask. Each byte represents one row, with bits 4..0 being pixels
// left-to-right. Only the digits 2-9 and uppercase letters used by
// captchaCharset are populated.
var glyphBitmaps = map[byte][7]byte{
	'2': {0b01110, 0b10001, 0b00001, 0b00010, 0b00100, 0b01000, 0b11111},
	'3': {0b01110, 0b10001, 0b00001, 0b00110, 0b00001, 0b10001, 0b01110},
	'4': {0b00010, 0b00110, 0b01010, 0b10010, 0b11111, 0b00010, 0b00010},
	'5': {0b11111, 0b10000, 0b11110, 0b00001, 0b00001, 0b10001, 0b01110},
	'6': {0b00110, 0b01000, 0b10000, 0b11110, 0b10001, 0b10001, 0b01110},
	'7': {0b11111, 0b00001, 0b00010, 0b00100, 0b01000, 0b01000, 0b01000},
	'8': {0b01110, 0b10001, 0b10001, 0b01110, 0b10001, 0b10001, 0b01110},
	'9': {0b01110, 0b10001, 0b10001, 0b01111, 0b00001, 0b00010, 0b01100},
	'A': {0b01110, 0b10001, 0b10001, 0b11111, 0b10001, 0b10001, 0b10001},
	'B': {0b11110, 0b10001, 0b10001, 0b11110, 0b10001, 0b10001, 0b11110},
	'C': {0b01110, 0b10001, 0b10000, 0b10000, 0b10000, 0b10001, 0b01110},
	'D': {0b11110, 0b10001, 0b10001, 0b10001, 0b10001, 0b10001, 0b11110},
	'E': {0b11111, 0b10000, 0b10000, 0b11110, 0b10000, 0b10000, 0b11111},
	'F': {0b11111, 0b10000, 0b10000, 0b11110, 0b10000, 0b10000, 0b10000},
	'G': {0b01110, 0b10001, 0b10000, 0b10111, 0b10001, 0b10001, 0b01111},
	'H': {0b10001, 0b10001, 0b10001, 0b11111, 0b10001, 0b10001, 0b10001},
	'J': {0b00001, 0b00001, 0b00001, 0b00001, 0b00001, 0b10001, 0b01110},
	'K': {0b10001, 0b10010, 0b10100, 0b11000, 0b10100, 0b10010, 0b10001},
	'L': {0b10000, 0b10000, 0b10000, 0b10000, 0b10000, 0b10000, 0b11111},
	'M': {0b10001, 0b11011, 0b10101, 0b10001, 0b10001, 0b10001, 0b10001},
	'N': {0b10001, 0b11001, 0b10101, 0b10011, 0b10001, 0b10001, 0b10001},
	'P': {0b11110, 0b10001, 0b10001, 0b11110, 0b10000, 0b10000, 0b10000},
	'Q': {0b01110, 0b10001, 0b10001, 0b10001, 0b10101, 0b10010, 0b01101},
	'R': {0b11110, 0b10001, 0b10001, 0b11110, 0b10100, 0b10010, 0b10001},
	'S': {0b01111, 0b10000, 0b10000, 0b01110, 0b00001, 0b00001, 0b11110},
	'T': {0b11111, 0b00100, 0b00100, 0b00100, 0b00100, 0b00100, 0b00100},
	'U': {0b10001, 0b10001, 0b10001, 0b10001, 0b10001, 0b10001, 0b01110},
	'V': {0b10001, 0b10001, 0b10001, 0b10001, 0b10001, 0b01010, 0b00100},
	'W': {0b10001, 0b10001, 0b10001, 0b10001, 0b10101, 0b11011, 0b10001},
	'X': {0b10001, 0b10001, 0b01010, 0b00100, 0b01010, 0b10001, 0b10001},
	'Y': {0b10001, 0b10001, 0b10001, 0b01010, 0b00100, 0b00100, 0b00100},
	'Z': {0b11111, 0b00001, 0b00010, 0b00100, 0b01000, 0b10000, 0b11111},
}
