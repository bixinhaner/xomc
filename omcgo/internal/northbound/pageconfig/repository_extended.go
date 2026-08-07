package pageconfig

import (
	"context"
	"crypto/subtle"
	"database/sql"
	"encoding/json"
	"fmt"
	"net"
	"strings"

	"github.com/jackc/pgx/v5"

	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
)

func (r *PgRepository) EnsureExtendedDefaults(ctx context.Context) error {
	return r.extDefaultsSeeded.Do(func() error {
		if err := r.ensureExtendedSchema(ctx); err != nil {
			return err
		}
		return r.seedExtendedDefaults(ctx)
	})
}

func (r *PgRepository) ensureExtendedSchema(ctx context.Context) error {
	statements := []string{
		`ALTER TABLE northbound_socket_alarm_configs ADD COLUMN IF NOT EXISTS max_clients integer DEFAULT 20 NOT NULL`,
		`ALTER TABLE northbound_socket_alarm_configs ADD COLUMN IF NOT EXISTS heartbeat_times integer DEFAULT 3 NOT NULL`,
		`ALTER TABLE northbound_socket_alarm_configs ALTER COLUMN heartbeat_seconds SET DEFAULT 60`,
		`DO $$
BEGIN
  ALTER TABLE northbound_socket_alarm_configs
    ADD CONSTRAINT northbound_socket_alarm_configs_max_clients_check
    CHECK (max_clients >= 1 AND max_clients <= 10000);
EXCEPTION WHEN duplicate_object THEN NULL;
END $$`,
		`DO $$
BEGIN
  ALTER TABLE northbound_socket_alarm_configs
    ADD CONSTRAINT northbound_socket_alarm_configs_heartbeat_times_check
    CHECK (heartbeat_times >= 1 AND heartbeat_times <= 100);
EXCEPTION WHEN duplicate_object THEN NULL;
END $$`,
		`UPDATE northbound_snmp_alarm_targets
		    SET priv_protocol = 'AES128'
		  WHERE version = 'v3' AND upper(priv_protocol) = 'AES'`,
		`UPDATE northbound_snmp_alarm_targets
		    SET mib_query_enabled = false
		  WHERE version = 'v3' AND mib_query_enabled = true`,
	}
	for _, statement := range statements {
		if _, err := r.pool.Exec(ctx, statement); err != nil {
			return fmt.Errorf("ensure northbound page-config schema: %w", err)
		}
	}
	return nil
}

func (r *PgRepository) seedExtendedDefaults(ctx context.Context) error {
	for _, target := range defaultDeliveryTargets() {
		if err := r.insertDefaultDeliveryTarget(ctx, target); err != nil {
			return err
		}
	}
	for _, target := range defaultSNMPAlarmTargets() {
		if err := r.insertDefaultSNMPAlarmTarget(ctx, target); err != nil {
			return err
		}
	}
	for _, config := range defaultSocketAlarmConfigs() {
		if err := r.insertDefaultSocketAlarmConfig(ctx, config); err != nil {
			return err
		}
	}
	for _, config := range defaultAPIConfigs() {
		if err := r.insertDefaultAPIConfig(ctx, config); err != nil {
			return err
		}
	}
	return nil
}

func (r *PgRepository) insertDefaultDeliveryTarget(ctx context.Context, target DeliveryTarget) error {
	target = normalizeDeliveryTarget(target)
	_, err := r.pool.Exec(ctx, `
INSERT INTO northbound_delivery_targets (
  scope, owner_code, target_key, name, enabled, protocol, host, port, username,
  auth_mode, remote_root, retry_times, timeout_seconds, passive_mode,
  host_key_policy, host_key_fingerprint
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16)
ON CONFLICT (scope, owner_code, target_key) DO NOTHING`,
		target.Scope, target.OwnerCode, target.Key, target.Name, target.Enabled, target.Protocol,
		target.Host, target.Port, target.Username, target.AuthMode, target.RemoteRoot,
		target.RetryTimes, target.TimeoutSeconds, target.PassiveMode, target.HostKeyPolicy,
		target.HostKeyFingerprint)
	if err != nil {
		return fmt.Errorf("insert default northbound_delivery_targets %s: %w", target.Key, err)
	}
	return nil
}

func (r *PgRepository) insertDefaultSNMPAlarmTarget(ctx context.Context, target SNMPAlarmTarget) error {
	fields, err := json.Marshal(target.MIBFields)
	if err != nil {
		return fmt.Errorf("marshal default SNMP MIB fields: %w", err)
	}
	_, err = r.pool.Exec(ctx, `
INSERT INTO northbound_snmp_alarm_targets (
  target_key, name, enabled, version, notification_type, listen_ip, listen_port,
  target_host, target_port, security_name, auth_protocol, priv_protocol,
  clear_severity_policy, mib_query_enabled, timeout_seconds, retries, mib_fields
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17)
ON CONFLICT (target_key) DO NOTHING`,
		target.Key, target.Name, target.Enabled, target.Version, target.NotificationType,
		target.ListenIP, target.ListenPort, target.TargetHost, target.TargetPort,
		target.SecurityName, target.AuthProtocol, target.PrivProtocol, target.ClearSeverityPolicy,
		target.MIBQueryEnabled, target.TimeoutSeconds, target.Retries, fields)
	if err != nil {
		return fmt.Errorf("insert default northbound_snmp_alarm_targets %s: %w", target.Key, err)
	}
	return nil
}

func (r *PgRepository) insertDefaultSocketAlarmConfig(ctx context.Context, config SocketAlarmConfig) error {
	accounts, err := json.Marshal(config.Accounts)
	if err != nil {
		return fmt.Errorf("marshal default socket accounts: %w", err)
	}
	_, err = r.pool.Exec(ctx, `
INSERT INTO northbound_socket_alarm_configs (
  config_key, name, enabled, profile, mode, listen_ip, listen_port,
  max_clients, realtime_push_enabled, client_sync_enabled, heartbeat_seconds,
  heartbeat_times,
  idle_timeout_seconds, accounts
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)
ON CONFLICT (config_key) DO NOTHING`,
		config.Key, config.Name, config.Enabled, config.Profile, "server", config.ListenIP,
		config.ListenPort, config.MaxClients, config.RealtimePushEnabled, config.ClientSyncEnabled,
		config.HeartbeatSeconds, config.HeartbeatTimes, config.IdleTimeoutSeconds, accounts)
	if err != nil {
		return fmt.Errorf("insert default northbound_socket_alarm_configs %s: %w", config.Key, err)
	}
	return nil
}

func (r *PgRepository) insertDefaultAPIConfig(ctx context.Context, config APIConfig) error {
	contract, err := json.Marshal(config.ResponseContract)
	if err != nil {
		return fmt.Errorf("marshal default API response contract: %w", err)
	}
	_, err = r.pool.Exec(ctx, `
INSERT INTO northbound_api_configs (
  api_key, name, method, path, kind, data_type, enabled,
  old_system_supported, current_supported, source, response_contract
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
ON CONFLICT (api_key) DO NOTHING`,
		config.Key, config.Name, config.Method, config.Path, config.Kind, config.DataType,
		config.Enabled, config.OldSystemSupported, config.CurrentSupported, config.Source, contract)
	if err != nil {
		return fmt.Errorf("insert default northbound_api_configs %s: %w", config.Key, err)
	}
	return nil
}

func (r *PgRepository) ListDeliveryTargets(ctx context.Context, filter DeliveryTargetFilter) ([]DeliveryTarget, error) {
	args := make([]any, 0, 2)
	where := []string{"true"}
	if filter.Scope != "" {
		args = append(args, filter.Scope)
		where = append(where, fmt.Sprintf("scope = $%d", len(args)))
	}
	if filter.OwnerCode != "" {
		args = append(args, filter.OwnerCode)
		where = append(where, fmt.Sprintf("owner_code = $%d", len(args)))
	}
	rows, err := r.pool.Query(ctx, fmt.Sprintf(`
SELECT id::text, scope, owner_code, target_key, name, enabled, protocol, host, port,
       username, credential_secret <> '' AS credential_set, auth_mode, remote_root,
       retry_times, timeout_seconds, passive_mode, host_key_policy,
       host_key_fingerprint, created_at, updated_at
  FROM northbound_delivery_targets
 WHERE %s
 ORDER BY scope ASC, owner_code ASC, target_key ASC`, strings.Join(where, " AND ")), args...)
	if err != nil {
		return nil, fmt.Errorf("query northbound_delivery_targets: %w", err)
	}
	defer rows.Close()
	out := make([]DeliveryTarget, 0)
	for rows.Next() {
		target, err := scanDeliveryTarget(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *target)
	}
	return out, rows.Err()
}

func (r *PgRepository) ListActiveDeliveryTargets(ctx context.Context, scope DeliveryScope, ownerCode string) ([]DeliveryTarget, error) {
	rows, err := r.pool.Query(ctx, `
SELECT id::text, scope, owner_code, target_key, name, enabled, protocol, host, port,
       username, credential_secret <> '' AS credential_set, auth_mode, remote_root,
       retry_times, timeout_seconds, passive_mode, host_key_policy,
       host_key_fingerprint, created_at, updated_at,
       credential_secret
  FROM northbound_delivery_targets
 WHERE scope=$1 AND owner_code=$2 AND enabled = true
 ORDER BY target_key ASC`, scope, ownerCode)
	if err != nil {
		return nil, fmt.Errorf("query active northbound_delivery_targets: %w", err)
	}
	defer rows.Close()

	out := make([]DeliveryTarget, 0)
	for rows.Next() {
		target, err := scanDeliveryTargetWithCredential(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *target)
	}
	return out, rows.Err()
}

func (r *PgRepository) ReplaceDeliveryTargets(ctx context.Context, req ReplaceDeliveryTargetsRequest) ([]DeliveryTarget, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin replace delivery targets: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	existingSecrets, err := loadDeliverySecrets(ctx, tx, req.Scope, req.OwnerCode)
	if err != nil {
		return nil, err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM northbound_delivery_targets WHERE scope=$1 AND owner_code=$2`, req.Scope, req.OwnerCode); err != nil {
		return nil, fmt.Errorf("delete northbound_delivery_targets: %w", err)
	}

	out := make([]DeliveryTarget, 0, len(req.Items))
	for _, item := range req.Items {
		item.Scope = req.Scope
		item.OwnerCode = req.OwnerCode
		item = normalizeDeliveryTarget(item)
		secret := credentialToPersist(item.Credential, existingSecrets[item.Key])
		row := tx.QueryRow(ctx, `
INSERT INTO northbound_delivery_targets (
  scope, owner_code, target_key, name, enabled, protocol, host, port, username,
  credential_secret, auth_mode, remote_root, retry_times, timeout_seconds,
  passive_mode, host_key_policy, host_key_fingerprint
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17)
RETURNING id::text, scope, owner_code, target_key, name, enabled, protocol, host, port,
          username, credential_secret <> '' AS credential_set, auth_mode, remote_root,
          retry_times, timeout_seconds, passive_mode, host_key_policy,
          host_key_fingerprint, created_at, updated_at`,
			item.Scope, item.OwnerCode, item.Key, item.Name, item.Enabled, item.Protocol,
			item.Host, item.Port, item.Username, secret, item.AuthMode, item.RemoteRoot,
			item.RetryTimes, item.TimeoutSeconds, item.PassiveMode, item.HostKeyPolicy,
			item.HostKeyFingerprint)
		created, err := scanDeliveryTarget(row)
		if err != nil {
			return nil, err
		}
		out = append(out, *created)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit replace delivery targets: %w", err)
	}
	return out, nil
}

func (r *PgRepository) ListSNMPAlarmTargets(ctx context.Context) ([]SNMPAlarmTarget, error) {
	rows, err := r.pool.Query(ctx, `
SELECT id::text, target_key, name, enabled, version, notification_type, listen_ip,
       listen_port, target_host, target_port, community_secret <> '' AS community_set,
       security_name, auth_protocol, auth_secret <> '' AS auth_credential_set,
       priv_protocol, priv_secret <> '' AS priv_credential_set, clear_severity_policy,
       mib_query_enabled, timeout_seconds, retries, mib_fields, created_at, updated_at
  FROM northbound_snmp_alarm_targets
 ORDER BY target_key ASC`)
	if err != nil {
		return nil, fmt.Errorf("query northbound_snmp_alarm_targets: %w", err)
	}
	defer rows.Close()
	out := make([]SNMPAlarmTarget, 0)
	for rows.Next() {
		target, err := scanSNMPAlarmTarget(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *target)
	}
	return out, rows.Err()
}

func (r *PgRepository) GetSNMPAlarmTargetForSend(ctx context.Context, key string) (*SNMPAlarmTarget, error) {
	row := r.pool.QueryRow(ctx, `
SELECT id::text, target_key, name, enabled, version, notification_type, listen_ip,
       listen_port, target_host, target_port, community_secret <> '' AS community_set,
       security_name, auth_protocol, auth_secret <> '' AS auth_credential_set,
       priv_protocol, priv_secret <> '' AS priv_credential_set, clear_severity_policy,
       mib_query_enabled, timeout_seconds, retries, mib_fields, created_at, updated_at,
       community_secret, auth_secret, priv_secret
  FROM northbound_snmp_alarm_targets
 WHERE target_key=$1`, key)
	return scanSNMPAlarmTargetWithSecrets(row)
}

func (r *PgRepository) ListActiveSNMPAlarmTargetsForSend(ctx context.Context) ([]SNMPAlarmTarget, error) {
	rows, err := r.pool.Query(ctx, `
SELECT id::text, target_key, name, enabled, version, notification_type, listen_ip,
       listen_port, target_host, target_port, community_secret <> '' AS community_set,
       security_name, auth_protocol, auth_secret <> '' AS auth_credential_set,
       priv_protocol, priv_secret <> '' AS priv_credential_set, clear_severity_policy,
       mib_query_enabled, timeout_seconds, retries, mib_fields, created_at, updated_at,
       community_secret, auth_secret, priv_secret
  FROM northbound_snmp_alarm_targets
 WHERE enabled = true
 ORDER BY target_key ASC`)
	if err != nil {
		return nil, fmt.Errorf("query active northbound_snmp_alarm_targets: %w", err)
	}
	defer rows.Close()
	out := make([]SNMPAlarmTarget, 0)
	for rows.Next() {
		target, err := scanSNMPAlarmTargetWithSecrets(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *target)
	}
	return out, rows.Err()
}

func (r *PgRepository) UpdateSNMPAlarmTarget(ctx context.Context, key string, target SNMPAlarmTarget) (*SNMPAlarmTarget, error) {
	current, _ := r.getSNMPAlarmSecrets(ctx, key)
	target = normalizeSNMPAlarmTarget(key, target)
	fields, err := json.Marshal(target.MIBFields)
	if err != nil {
		return nil, fmt.Errorf("marshal SNMP MIB fields: %w", err)
	}
	row := r.pool.QueryRow(ctx, `
INSERT INTO northbound_snmp_alarm_targets (
  target_key, name, enabled, version, notification_type, listen_ip, listen_port,
  target_host, target_port, community_secret, security_name, auth_protocol,
  auth_secret, priv_protocol, priv_secret, clear_severity_policy, mib_query_enabled,
  timeout_seconds, retries, mib_fields
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20)
ON CONFLICT (target_key) DO UPDATE SET
  name=EXCLUDED.name, enabled=EXCLUDED.enabled, version=EXCLUDED.version,
  notification_type=EXCLUDED.notification_type, listen_ip=EXCLUDED.listen_ip,
  listen_port=EXCLUDED.listen_port, target_host=EXCLUDED.target_host,
  target_port=EXCLUDED.target_port, community_secret=EXCLUDED.community_secret,
  security_name=EXCLUDED.security_name, auth_protocol=EXCLUDED.auth_protocol,
  auth_secret=EXCLUDED.auth_secret, priv_protocol=EXCLUDED.priv_protocol,
  priv_secret=EXCLUDED.priv_secret, clear_severity_policy=EXCLUDED.clear_severity_policy,
  mib_query_enabled=EXCLUDED.mib_query_enabled, timeout_seconds=EXCLUDED.timeout_seconds,
  retries=EXCLUDED.retries, mib_fields=EXCLUDED.mib_fields
RETURNING id::text, target_key, name, enabled, version, notification_type, listen_ip,
          listen_port, target_host, target_port, community_secret <> '' AS community_set,
          security_name, auth_protocol, auth_secret <> '' AS auth_credential_set,
          priv_protocol, priv_secret <> '' AS priv_credential_set, clear_severity_policy,
          mib_query_enabled, timeout_seconds, retries, mib_fields, created_at, updated_at`,
		target.Key, target.Name, target.Enabled, target.Version, target.NotificationType,
		target.ListenIP, target.ListenPort, target.TargetHost, target.TargetPort,
		credentialToPersist(target.Community, current.community), target.SecurityName, target.AuthProtocol,
		credentialToPersist(target.AuthCredential, current.auth), target.PrivProtocol,
		credentialToPersist(target.PrivCredential, current.priv), target.ClearSeverityPolicy,
		target.MIBQueryEnabled, target.TimeoutSeconds, target.Retries, fields)
	return scanSNMPAlarmTarget(row)
}

func (r *PgRepository) ListSocketAlarmConfigs(ctx context.Context) ([]SocketAlarmConfig, error) {
	rows, err := r.pool.Query(ctx, `
SELECT id::text, config_key, name, enabled, profile, mode, listen_ip, listen_port,
       max_clients, realtime_push_enabled, client_sync_enabled, heartbeat_seconds,
       heartbeat_times,
       idle_timeout_seconds, accounts, created_at, updated_at
  FROM northbound_socket_alarm_configs
 ORDER BY config_key ASC`)
	if err != nil {
		return nil, fmt.Errorf("query northbound_socket_alarm_configs: %w", err)
	}
	defer rows.Close()
	out := make([]SocketAlarmConfig, 0)
	for rows.Next() {
		config, err := scanSocketAlarmConfig(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *config)
	}
	return out, rows.Err()
}

func (r *PgRepository) ListActiveSocketAlarmConfigsForServe(ctx context.Context) ([]SocketAlarmConfig, error) {
	rows, err := r.pool.Query(ctx, `
SELECT id::text, config_key, name, enabled, profile, mode, listen_ip, listen_port,
       max_clients, realtime_push_enabled, client_sync_enabled, heartbeat_seconds,
       heartbeat_times,
       idle_timeout_seconds, accounts, created_at, updated_at
  FROM northbound_socket_alarm_configs
 WHERE enabled = true AND mode = 'server'
 ORDER BY config_key ASC`)
	if err != nil {
		return nil, fmt.Errorf("query active northbound_socket_alarm_configs: %w", err)
	}
	defer rows.Close()
	out := make([]SocketAlarmConfig, 0)
	for rows.Next() {
		config, err := scanSocketAlarmConfigWithSecrets(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *config)
	}
	return out, rows.Err()
}

func (r *PgRepository) UpdateSocketAlarmConfig(ctx context.Context, key string, config SocketAlarmConfig) (*SocketAlarmConfig, error) {
	currentSecrets, _ := r.getSocketAccountSecrets(ctx, key)
	config = normalizeSocketAlarmConfig(key, config)
	config.Accounts = mergeSocketAccountSecrets(config.Accounts, currentSecrets)
	accounts, err := json.Marshal(config.Accounts)
	if err != nil {
		return nil, fmt.Errorf("marshal socket accounts: %w", err)
	}
	row := r.pool.QueryRow(ctx, `
INSERT INTO northbound_socket_alarm_configs (
  config_key, name, enabled, profile, mode, listen_ip, listen_port,
  max_clients, realtime_push_enabled, client_sync_enabled, heartbeat_seconds,
  heartbeat_times,
  idle_timeout_seconds, accounts
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)
ON CONFLICT (config_key) DO UPDATE SET
  name=EXCLUDED.name, enabled=EXCLUDED.enabled, profile=EXCLUDED.profile,
  mode=EXCLUDED.mode, listen_ip=EXCLUDED.listen_ip, listen_port=EXCLUDED.listen_port,
  max_clients=EXCLUDED.max_clients,
  realtime_push_enabled=EXCLUDED.realtime_push_enabled,
  client_sync_enabled=EXCLUDED.client_sync_enabled,
  heartbeat_seconds=EXCLUDED.heartbeat_seconds,
  heartbeat_times=EXCLUDED.heartbeat_times,
  idle_timeout_seconds=EXCLUDED.idle_timeout_seconds,
  accounts=EXCLUDED.accounts
RETURNING id::text, config_key, name, enabled, profile, mode, listen_ip, listen_port,
          max_clients, realtime_push_enabled, client_sync_enabled, heartbeat_seconds,
          heartbeat_times,
          idle_timeout_seconds, accounts, created_at, updated_at`,
		config.Key, config.Name, config.Enabled, config.Profile, config.Mode, config.ListenIP,
		config.ListenPort, config.MaxClients, config.RealtimePushEnabled, config.ClientSyncEnabled,
		config.HeartbeatSeconds, config.HeartbeatTimes, config.IdleTimeoutSeconds, accounts)
	return scanSocketAlarmConfig(row)
}

func (r *PgRepository) ListAPIConfigs(ctx context.Context) ([]APIConfig, error) {
	rows, err := r.pool.Query(ctx, `
SELECT id::text, api_key, name, method, path, kind, data_type, enabled,
       old_system_supported, current_supported, source, response_contract,
       created_at, updated_at
  FROM northbound_api_configs
 WHERE old_system_supported = true AND current_supported = true
 ORDER BY api_key ASC`)
	if err != nil {
		return nil, fmt.Errorf("query northbound_api_configs: %w", err)
	}
	defer rows.Close()
	out := make([]APIConfig, 0)
	for rows.Next() {
		config, err := scanAPIConfig(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *config)
	}
	return out, rows.Err()
}

func (r *PgRepository) UpdateAPIConfig(ctx context.Context, key string, enabled bool) (*APIConfig, error) {
	row := r.pool.QueryRow(ctx, `
UPDATE northbound_api_configs
   SET enabled=$2
 WHERE api_key=$1 AND old_system_supported = true AND current_supported = true
RETURNING id::text, api_key, name, method, path, kind, data_type, enabled,
          old_system_supported, current_supported, source, response_contract,
          created_at, updated_at`, key, enabled)
	return scanAPIConfig(row)
}

func (r *PgRepository) ListAPIClients(ctx context.Context) ([]APIClient, error) {
	rows, err := r.pool.Query(ctx, `
SELECT id::text, client_key, name, enabled, token_secret <> '' AS token_set,
       allowed_api_keys, ip_whitelist, expires_at, created_at, updated_at
  FROM northbound_api_clients
 ORDER BY client_key ASC`)
	if err != nil {
		return nil, fmt.Errorf("query northbound_api_clients: %w", err)
	}
	defer rows.Close()
	out := make([]APIClient, 0)
	for rows.Next() {
		client, err := scanAPIClient(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *client)
	}
	return out, rows.Err()
}

func (r *PgRepository) ReplaceAPIClients(ctx context.Context, req ReplaceAPIClientsRequest) ([]APIClient, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin replace API clients: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	existingSecrets, err := loadAPIClientSecrets(ctx, tx)
	if err != nil {
		return nil, err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM northbound_api_clients`); err != nil {
		return nil, fmt.Errorf("delete northbound_api_clients: %w", err)
	}

	out := make([]APIClient, 0, len(req.Items))
	for _, item := range req.Items {
		item = normalizeAPIClient(item)
		item.TokenSecret = credentialToPersist(item.TokenSecret, existingSecrets[item.ClientKey])
		if err := validateAPIClient(item); err != nil {
			return nil, err
		}
		allowed, err := json.Marshal(item.AllowedAPIKeys)
		if err != nil {
			return nil, fmt.Errorf("marshal API client allowed_api_keys: %w", err)
		}
		whitelist, err := json.Marshal(item.IPWhitelist)
		if err != nil {
			return nil, fmt.Errorf("marshal API client ip_whitelist: %w", err)
		}
		row := tx.QueryRow(ctx, `
INSERT INTO northbound_api_clients (
  client_key, name, enabled, token_secret, allowed_api_keys, ip_whitelist, expires_at
) VALUES ($1,$2,$3,$4,$5,$6,$7)
RETURNING id::text, client_key, name, enabled, token_secret <> '' AS token_set,
          allowed_api_keys, ip_whitelist, expires_at, created_at, updated_at`,
			item.ClientKey, item.Name, item.Enabled, item.TokenSecret, allowed, whitelist, item.ExpiresAt)
		created, err := scanAPIClient(row)
		if err != nil {
			return nil, err
		}
		out = append(out, *created)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit replace API clients: %w", err)
	}
	return out, nil
}

func (r *PgRepository) AuthenticateAPIClient(ctx context.Context, credential string, remoteIP string, apiKey string) (*APIClient, error) {
	var activeCount int
	if err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM northbound_api_clients WHERE enabled = true`).Scan(&activeCount); err != nil {
		return nil, fmt.Errorf("count active northbound_api_clients: %w", err)
	}
	if activeCount == 0 {
		return nil, nil
	}
	credential = strings.TrimSpace(credential)
	if credential == "" {
		return nil, fmt.Errorf("%w: northbound API client token is required", commonerrors.ErrUnauthorized)
	}

	rows, err := r.pool.Query(ctx, `
SELECT id::text, client_key, name, enabled, token_secret, allowed_api_keys,
       ip_whitelist, expires_at, created_at, updated_at
  FROM northbound_api_clients
 WHERE enabled = true
   AND (expires_at IS NULL OR expires_at > now())
 ORDER BY client_key ASC`)
	if err != nil {
		return nil, fmt.Errorf("query active northbound_api_clients: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		client, token, err := scanAPIClientWithSecret(rows)
		if err != nil {
			return nil, err
		}
		if subtle.ConstantTimeCompare([]byte(token), []byte(credential)) != 1 {
			continue
		}
		if !apiClientAllowsAPI(*client, apiKey) {
			return nil, fmt.Errorf("%w: northbound API client %s cannot access %s", commonerrors.ErrForbidden, client.ClientKey, apiKey)
		}
		if !apiClientAllowsIP(*client, remoteIP) {
			return nil, fmt.Errorf("%w: northbound API client %s IP %s is not whitelisted", commonerrors.ErrForbidden, client.ClientKey, remoteIP)
		}
		client.TokenSecret = ""
		client.TokenSet = true
		return client, nil
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return nil, fmt.Errorf("%w: invalid northbound API client token", commonerrors.ErrUnauthorized)
}

func scanDeliveryTarget(row scanner) (*DeliveryTarget, error) {
	var target DeliveryTarget
	var scope, protocol, authMode, hostKeyPolicy string
	if err := row.Scan(
		&target.ID, &scope, &target.OwnerCode, &target.Key, &target.Name, &target.Enabled,
		&protocol, &target.Host, &target.Port, &target.Username, &target.CredentialSet,
		&authMode, &target.RemoteRoot, &target.RetryTimes, &target.TimeoutSeconds,
		&target.PassiveMode, &hostKeyPolicy, &target.HostKeyFingerprint,
		&target.CreatedAt, &target.UpdatedAt,
	); err != nil {
		if err == pgx.ErrNoRows {
			return nil, commonerrors.ErrNotFound
		}
		return nil, fmt.Errorf("scan northbound_delivery_targets row: %w", err)
	}
	target.Scope = DeliveryScope(scope)
	target.Protocol = DeliveryProtocol(protocol)
	target.AuthMode = DeliveryAuthMode(authMode)
	target.HostKeyPolicy = DeliveryHostKeyPolicy(hostKeyPolicy)
	return &target, nil
}

func scanDeliveryTargetWithCredential(row scanner) (*DeliveryTarget, error) {
	var target DeliveryTarget
	var scope, protocol, authMode, hostKeyPolicy string
	var credential string
	if err := row.Scan(
		&target.ID, &scope, &target.OwnerCode, &target.Key, &target.Name, &target.Enabled,
		&protocol, &target.Host, &target.Port, &target.Username, &target.CredentialSet,
		&authMode, &target.RemoteRoot, &target.RetryTimes, &target.TimeoutSeconds,
		&target.PassiveMode, &hostKeyPolicy, &target.HostKeyFingerprint,
		&target.CreatedAt, &target.UpdatedAt, &credential,
	); err != nil {
		if err == pgx.ErrNoRows {
			return nil, commonerrors.ErrNotFound
		}
		return nil, fmt.Errorf("scan active northbound_delivery_targets row: %w", err)
	}
	target.Scope = DeliveryScope(scope)
	target.Protocol = DeliveryProtocol(protocol)
	target.AuthMode = DeliveryAuthMode(authMode)
	target.HostKeyPolicy = DeliveryHostKeyPolicy(hostKeyPolicy)
	target.Credential = credential
	return &target, nil
}

func scanSNMPAlarmTarget(row scanner) (*SNMPAlarmTarget, error) {
	var target SNMPAlarmTarget
	var fieldsRaw []byte
	if err := row.Scan(
		&target.ID, &target.Key, &target.Name, &target.Enabled, &target.Version,
		&target.NotificationType, &target.ListenIP, &target.ListenPort, &target.TargetHost,
		&target.TargetPort, &target.CommunitySet, &target.SecurityName, &target.AuthProtocol,
		&target.AuthCredentialSet, &target.PrivProtocol, &target.PrivCredentialSet,
		&target.ClearSeverityPolicy, &target.MIBQueryEnabled, &target.TimeoutSeconds,
		&target.Retries, &fieldsRaw, &target.CreatedAt, &target.UpdatedAt,
	); err != nil {
		if err == pgx.ErrNoRows {
			return nil, commonerrors.ErrNotFound
		}
		return nil, fmt.Errorf("scan northbound_snmp_alarm_targets row: %w", err)
	}
	if len(fieldsRaw) > 0 {
		if err := json.Unmarshal(fieldsRaw, &target.MIBFields); err != nil {
			return nil, fmt.Errorf("unmarshal SNMP MIB fields: %w", err)
		}
	}
	if len(target.MIBFields) == 0 {
		target.MIBFields = defaultSNMPAlarmFields()
	}
	return &target, nil
}

func scanSNMPAlarmTargetWithSecrets(row scanner) (*SNMPAlarmTarget, error) {
	var target SNMPAlarmTarget
	var fieldsRaw []byte
	var community, authCredential, privCredential string
	if err := row.Scan(
		&target.ID, &target.Key, &target.Name, &target.Enabled, &target.Version,
		&target.NotificationType, &target.ListenIP, &target.ListenPort, &target.TargetHost,
		&target.TargetPort, &target.CommunitySet, &target.SecurityName, &target.AuthProtocol,
		&target.AuthCredentialSet, &target.PrivProtocol, &target.PrivCredentialSet,
		&target.ClearSeverityPolicy, &target.MIBQueryEnabled, &target.TimeoutSeconds,
		&target.Retries, &fieldsRaw, &target.CreatedAt, &target.UpdatedAt,
		&community, &authCredential, &privCredential,
	); err != nil {
		if err == pgx.ErrNoRows {
			return nil, commonerrors.ErrNotFound
		}
		return nil, fmt.Errorf("scan northbound_snmp_alarm_targets send row: %w", err)
	}
	if len(fieldsRaw) > 0 {
		if err := json.Unmarshal(fieldsRaw, &target.MIBFields); err != nil {
			return nil, fmt.Errorf("unmarshal SNMP MIB fields: %w", err)
		}
	}
	if len(target.MIBFields) == 0 {
		target.MIBFields = defaultSNMPAlarmFields()
	}
	target.Community = community
	target.AuthCredential = authCredential
	target.PrivCredential = privCredential
	return &target, nil
}

func scanSocketAlarmConfig(row scanner) (*SocketAlarmConfig, error) {
	var config SocketAlarmConfig
	var accountsRaw []byte
	if err := row.Scan(
		&config.ID, &config.Key, &config.Name, &config.Enabled, &config.Profile,
		&config.Mode, &config.ListenIP, &config.ListenPort, &config.MaxClients,
		&config.RealtimePushEnabled, &config.ClientSyncEnabled, &config.HeartbeatSeconds,
		&config.HeartbeatTimes, &config.IdleTimeoutSeconds,
		&accountsRaw, &config.CreatedAt, &config.UpdatedAt,
	); err != nil {
		if err == pgx.ErrNoRows {
			return nil, commonerrors.ErrNotFound
		}
		return nil, fmt.Errorf("scan northbound_socket_alarm_configs row: %w", err)
	}
	if len(accountsRaw) > 0 {
		if err := json.Unmarshal(accountsRaw, &config.Accounts); err != nil {
			return nil, fmt.Errorf("unmarshal socket accounts: %w", err)
		}
	}
	redactSocketAccountCredentials(config.Accounts)
	return &config, nil
}

func scanSocketAlarmConfigWithSecrets(row scanner) (*SocketAlarmConfig, error) {
	config, err := scanSocketAlarmConfigRaw(row)
	if err != nil {
		return nil, err
	}
	for i := range config.Accounts {
		config.Accounts[i].CredentialSet = strings.TrimSpace(config.Accounts[i].Credential) != ""
	}
	return config, nil
}

func scanSocketAlarmConfigRaw(row scanner) (*SocketAlarmConfig, error) {
	var config SocketAlarmConfig
	var accountsRaw []byte
	if err := row.Scan(
		&config.ID, &config.Key, &config.Name, &config.Enabled, &config.Profile,
		&config.Mode, &config.ListenIP, &config.ListenPort, &config.MaxClients,
		&config.RealtimePushEnabled, &config.ClientSyncEnabled, &config.HeartbeatSeconds,
		&config.HeartbeatTimes, &config.IdleTimeoutSeconds,
		&accountsRaw, &config.CreatedAt, &config.UpdatedAt,
	); err != nil {
		if err == pgx.ErrNoRows {
			return nil, commonerrors.ErrNotFound
		}
		return nil, fmt.Errorf("scan northbound_socket_alarm_configs row: %w", err)
	}
	if len(accountsRaw) > 0 {
		if err := json.Unmarshal(accountsRaw, &config.Accounts); err != nil {
			return nil, fmt.Errorf("unmarshal socket accounts: %w", err)
		}
	}
	return &config, nil
}

func scanAPIConfig(row scanner) (*APIConfig, error) {
	var config APIConfig
	var contractRaw []byte
	if err := row.Scan(
		&config.ID, &config.Key, &config.Name, &config.Method, &config.Path, &config.Kind,
		&config.DataType, &config.Enabled, &config.OldSystemSupported, &config.CurrentSupported,
		&config.Source, &contractRaw, &config.CreatedAt, &config.UpdatedAt,
	); err != nil {
		if err == pgx.ErrNoRows {
			return nil, commonerrors.ErrNotFound
		}
		return nil, fmt.Errorf("scan northbound_api_configs row: %w", err)
	}
	if len(contractRaw) > 0 {
		if err := json.Unmarshal(contractRaw, &config.ResponseContract); err != nil {
			return nil, fmt.Errorf("unmarshal API response contract: %w", err)
		}
	}
	return &config, nil
}

func scanAPIClient(row scanner) (*APIClient, error) {
	var client APIClient
	var allowedRaw, whitelistRaw []byte
	var expiresAt sql.NullTime
	if err := row.Scan(
		&client.ID, &client.ClientKey, &client.Name, &client.Enabled, &client.TokenSet,
		&allowedRaw, &whitelistRaw, &expiresAt, &client.CreatedAt, &client.UpdatedAt,
	); err != nil {
		if err == pgx.ErrNoRows {
			return nil, commonerrors.ErrNotFound
		}
		return nil, fmt.Errorf("scan northbound_api_clients row: %w", err)
	}
	if len(allowedRaw) > 0 {
		if err := json.Unmarshal(allowedRaw, &client.AllowedAPIKeys); err != nil {
			return nil, fmt.Errorf("unmarshal API client allowed_api_keys: %w", err)
		}
	}
	if len(whitelistRaw) > 0 {
		if err := json.Unmarshal(whitelistRaw, &client.IPWhitelist); err != nil {
			return nil, fmt.Errorf("unmarshal API client ip_whitelist: %w", err)
		}
	}
	if expiresAt.Valid {
		t := expiresAt.Time
		client.ExpiresAt = &t
	}
	return &client, nil
}

func scanAPIClientWithSecret(row scanner) (*APIClient, string, error) {
	var client APIClient
	var allowedRaw, whitelistRaw []byte
	var expiresAt sql.NullTime
	var token string
	if err := row.Scan(
		&client.ID, &client.ClientKey, &client.Name, &client.Enabled, &token,
		&allowedRaw, &whitelistRaw, &expiresAt, &client.CreatedAt, &client.UpdatedAt,
	); err != nil {
		if err == pgx.ErrNoRows {
			return nil, "", commonerrors.ErrNotFound
		}
		return nil, "", fmt.Errorf("scan active northbound_api_clients row: %w", err)
	}
	if len(allowedRaw) > 0 {
		if err := json.Unmarshal(allowedRaw, &client.AllowedAPIKeys); err != nil {
			return nil, "", fmt.Errorf("unmarshal API client allowed_api_keys: %w", err)
		}
	}
	if len(whitelistRaw) > 0 {
		if err := json.Unmarshal(whitelistRaw, &client.IPWhitelist); err != nil {
			return nil, "", fmt.Errorf("unmarshal API client ip_whitelist: %w", err)
		}
	}
	if expiresAt.Valid {
		t := expiresAt.Time
		client.ExpiresAt = &t
	}
	client.TokenSet = strings.TrimSpace(token) != ""
	return &client, token, nil
}

func loadDeliverySecrets(ctx context.Context, tx pgx.Tx, scope DeliveryScope, ownerCode string) (map[string]string, error) {
	rows, err := tx.Query(ctx, `SELECT target_key, credential_secret FROM northbound_delivery_targets WHERE scope=$1 AND owner_code=$2`, scope, ownerCode)
	if err != nil {
		return nil, fmt.Errorf("query delivery target secrets: %w", err)
	}
	defer rows.Close()
	out := map[string]string{}
	for rows.Next() {
		var key, secret string
		if err := rows.Scan(&key, &secret); err != nil {
			return nil, fmt.Errorf("scan delivery target secrets: %w", err)
		}
		out[key] = secret
	}
	return out, rows.Err()
}

func loadAPIClientSecrets(ctx context.Context, tx pgx.Tx) (map[string]string, error) {
	rows, err := tx.Query(ctx, `SELECT client_key, token_secret FROM northbound_api_clients`)
	if err != nil {
		return nil, fmt.Errorf("query API client secrets: %w", err)
	}
	defer rows.Close()
	out := map[string]string{}
	for rows.Next() {
		var key, secret string
		if err := rows.Scan(&key, &secret); err != nil {
			return nil, fmt.Errorf("scan API client secrets: %w", err)
		}
		out[key] = secret
	}
	return out, rows.Err()
}

type snmpSecrets struct {
	community string
	auth      string
	priv      string
}

func (r *PgRepository) getSNMPAlarmSecrets(ctx context.Context, key string) (snmpSecrets, error) {
	var out snmpSecrets
	err := r.pool.QueryRow(ctx, `SELECT community_secret, auth_secret, priv_secret FROM northbound_snmp_alarm_targets WHERE target_key=$1`, key).
		Scan(&out.community, &out.auth, &out.priv)
	if err != nil {
		if err == pgx.ErrNoRows {
			return out, nil
		}
		return out, fmt.Errorf("query SNMP target secrets: %w", err)
	}
	return out, nil
}

func (r *PgRepository) getSocketAccountSecrets(ctx context.Context, key string) (map[string]string, error) {
	var raw []byte
	if err := r.pool.QueryRow(ctx, `SELECT accounts FROM northbound_socket_alarm_configs WHERE config_key=$1`, key).Scan(&raw); err != nil {
		if err == pgx.ErrNoRows {
			return map[string]string{}, nil
		}
		return nil, fmt.Errorf("query socket account secrets: %w", err)
	}
	var accounts []SocketAccount
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &accounts); err != nil {
			return nil, fmt.Errorf("unmarshal socket account secrets: %w", err)
		}
	}
	out := map[string]string{}
	for _, account := range accounts {
		out[account.Key] = account.Credential
	}
	return out, nil
}

func normalizeDeliveryTarget(target DeliveryTarget) DeliveryTarget {
	if strings.TrimSpace(target.Name) == "" {
		target.Name = target.Key
	}
	if target.Protocol == "" {
		target.Protocol = DeliveryProtocolFTP
	}
	if target.Port == 0 {
		if target.Protocol == DeliveryProtocolSFTP {
			target.Port = 22
		} else {
			target.Port = 21
		}
	}
	if target.AuthMode == "" {
		target.AuthMode = DeliveryAuthPassword
	}
	if target.HostKeyPolicy == "" {
		target.HostKeyPolicy = DeliveryHostKeyInsecure
	}
	target.HostKeyFingerprint = strings.TrimSpace(target.HostKeyFingerprint)
	// FTP passive mode is the only implemented data-channel mode (active mode is rejected
	// at upload time); force it on so callers cannot disable it.
	target.PassiveMode = true
	if strings.TrimSpace(target.RemoteRoot) == "" {
		target.RemoteRoot = "/northupload"
	}
	if target.TimeoutSeconds == 0 {
		target.TimeoutSeconds = 30
	}
	return target
}

func normalizeSNMPAlarmTarget(key string, target SNMPAlarmTarget) SNMPAlarmTarget {
	if target.Key == "" {
		target.Key = key
	}
	if target.Name == "" {
		target.Name = target.Key
	}
	if target.Version == "" {
		target.Version = "v2"
	}
	if target.NotificationType == "" {
		target.NotificationType = "Trap"
	}
	if target.ListenIP == "" {
		target.ListenIP = "0.0.0.0"
	}
	if target.ListenPort == 0 {
		target.ListenPort = 161
	}
	if target.TargetPort == 0 {
		target.TargetPort = 162
	}
	if target.AuthProtocol == "" && strings.EqualFold(target.Version, "v3") {
		target.AuthProtocol = "SHA"
	}
	if target.PrivProtocol == "" && strings.EqualFold(target.Version, "v3") {
		target.PrivProtocol = "AES128"
	}
	if strings.EqualFold(target.Version, "v3") {
		target.MIBQueryEnabled = false
	}
	if target.ClearSeverityPolicy == "" {
		target.ClearSeverityPolicy = "保留原级别"
	}
	if target.TimeoutSeconds == 0 {
		target.TimeoutSeconds = 5
	}
	if len(target.MIBFields) == 0 {
		target.MIBFields = defaultSNMPAlarmFields()
	}
	return target
}

func normalizeSocketAlarmConfig(key string, config SocketAlarmConfig) SocketAlarmConfig {
	if config.Key == "" {
		config.Key = key
	}
	if config.Name == "" {
		config.Name = config.Key
	}
	if config.Mode == "" {
		config.Mode = "server"
	}
	if config.ListenIP == "" {
		config.ListenIP = "0.0.0.0"
	}
	if config.ListenPort == 0 {
		config.ListenPort = 31232
	}
	if config.MaxClients == 0 {
		config.MaxClients = 20
	}
	if config.HeartbeatSeconds == 0 {
		config.HeartbeatSeconds = 60
	}
	if config.HeartbeatTimes == 0 {
		config.HeartbeatTimes = 3
	}
	if config.IdleTimeoutSeconds == 0 {
		config.IdleTimeoutSeconds = config.HeartbeatSeconds * config.HeartbeatTimes
		if config.IdleTimeoutSeconds == 0 {
			config.IdleTimeoutSeconds = 180
		}
	}
	return config
}

func normalizeAPIClient(client APIClient) APIClient {
	client.ClientKey = strings.TrimSpace(client.ClientKey)
	client.Name = strings.TrimSpace(client.Name)
	if client.Name == "" {
		client.Name = client.ClientKey
	}
	client.TokenSecret = strings.TrimSpace(client.TokenSecret)
	client.AllowedAPIKeys = normalizeStringList(client.AllowedAPIKeys)
	client.IPWhitelist = normalizeStringList(client.IPWhitelist)
	return client
}

func validateAPIClient(client APIClient) error {
	if strings.TrimSpace(client.ClientKey) == "" {
		return fmt.Errorf("%w: API client_key is required", commonerrors.ErrInvalidInput)
	}
	if client.Enabled && strings.TrimSpace(client.TokenSecret) == "" {
		return fmt.Errorf("%w: API client token_secret is required when enabled", commonerrors.ErrInvalidInput)
	}
	for _, entry := range client.IPWhitelist {
		if net.ParseIP(entry) != nil {
			continue
		}
		if _, _, err := net.ParseCIDR(entry); err == nil {
			continue
		}
		return fmt.Errorf("%w: invalid API client ip_whitelist entry %s", commonerrors.ErrInvalidInput, entry)
	}
	return nil
}

func normalizeStringList(values []string) []string {
	out := make([]string, 0, len(values))
	seen := map[string]struct{}{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func apiClientAllowsAPI(client APIClient, apiKey string) bool {
	if len(client.AllowedAPIKeys) == 0 {
		return true
	}
	for _, allowed := range client.AllowedAPIKeys {
		if strings.EqualFold(allowed, apiKey) {
			return true
		}
	}
	return false
}

func apiClientAllowsIP(client APIClient, remoteIP string) bool {
	if len(client.IPWhitelist) == 0 {
		return true
	}
	host := strings.TrimSpace(remoteIP)
	if parsedHost, _, err := net.SplitHostPort(host); err == nil {
		host = parsedHost
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return false
	}
	for _, entry := range client.IPWhitelist {
		if allowed := net.ParseIP(entry); allowed != nil {
			if allowed.Equal(ip) {
				return true
			}
			continue
		}
		if _, network, err := net.ParseCIDR(entry); err == nil && network.Contains(ip) {
			return true
		}
	}
	return false
}

func credentialToPersist(next, current string) string {
	next = strings.TrimSpace(next)
	if next == "" || next == "已加密存储" || next == "待保存" {
		return current
	}
	return next
}

func mergeSocketAccountSecrets(accounts []SocketAccount, current map[string]string) []SocketAccount {
	for i := range accounts {
		accounts[i].Credential = credentialToPersist(accounts[i].Credential, current[accounts[i].Key])
	}
	return accounts
}

func redactSocketAccountCredentials(accounts []SocketAccount) {
	for i := range accounts {
		accounts[i].CredentialSet = strings.TrimSpace(accounts[i].Credential) != ""
		accounts[i].Credential = ""
	}
}
