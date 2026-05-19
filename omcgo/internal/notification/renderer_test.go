package notification

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRenderTemplate(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		subject     string
		body        string
		vars        map[string]string
		wantSubject string
		wantBody    string
		wantErr     bool
	}{
		{
			name:        "normal substitution",
			subject:     "告警: {{.alertname}}",
			body:        "设备 {{.device_sn}} 触发了 {{.alertname}}",
			vars:        map[string]string{"alertname": "OMCAppDown", "device_sn": "SN123"},
			wantSubject: "告警: OMCAppDown",
			wantBody:    "设备 SN123 触发了 OMCAppDown",
		},
		{
			name:        "missing var renders empty",
			subject:     "x{{.absent}}y",
			body:        "body",
			vars:        map[string]string{},
			wantSubject: "xy",
			wantBody:    "body",
		},
		{
			name:        "nil vars map",
			subject:     "static",
			body:        "{{.k}}-end",
			vars:        nil,
			wantSubject: "static",
			wantBody:    "-end",
		},
		{
			name:    "invalid template syntax in subject",
			subject: "{{.unclosed",
			body:    "ok",
			vars:    map[string]string{},
			wantErr: true,
		},
		{
			name:    "invalid template syntax in body",
			subject: "ok",
			body:    "{{.bad",
			vars:    map[string]string{},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tpl := &NotificationTemplate{Subject: tt.subject, Body: tt.body}
			subject, body, err := RenderTemplate(tpl, tt.vars)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.wantSubject, subject)
			require.Equal(t, tt.wantBody, body)
		})
	}
}

func TestRenderTemplate_NilTemplate(t *testing.T) {
	t.Parallel()
	_, _, err := RenderTemplate(nil, nil)
	require.Error(t, err)
}
