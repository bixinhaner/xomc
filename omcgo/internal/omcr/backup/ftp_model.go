package backup

import (
	"time"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/common/model"
)

// FTPConfig represents an FTP server configuration used for backup file transfers.
type FTPConfig struct {
	ID                uuid.UUID `json:"id"`
	ConfigName        string    `json:"config_name"`
	Host              string    `json:"host"`
	Port              int       `json:"port"`
	Username          string    `json:"username"`
	PasswordEncrypted *string   `json:"password_encrypted,omitempty"`
	Protocol          string    `json:"protocol"`
	RemotePath        string    `json:"remote_path"`
	Passive           bool      `json:"passive"`
	Enabled           bool      `json:"enabled"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

// FTPConfigFilter specifies criteria for listing FTP configurations.
type FTPConfigFilter struct {
	Enabled *bool
	model.ListRequest
}
