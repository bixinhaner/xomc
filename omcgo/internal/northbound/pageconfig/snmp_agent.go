package pageconfig

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	g "github.com/gosnmp/gosnmp"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/alarm"
	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/core/model"
	nbsnmp "github.com/omcgo/omcgo/internal/northbound/snmp"
)

const (
	snmpAgentCacheTTL  = 3 * time.Second
	snmpAgentMaxRows   = 10000
	snmpAgentPageSize  = 1000
	snmpMaxBulkRows    = 100
	snmpMaxOIDResponse = 65535
	snmpAgentEngineID  = "goomc-snmp-agent"

	oidUSMStatsUnknownEngineIDs = ".1.3.6.1.6.3.15.1.1.4.0"
)

var snmpAgentGosnmpLogger = g.NewLogger(log.New(io.Discard, "", 0))

type SNMPMIBAgentManager struct {
	svc    *Service
	logger *zap.Logger

	mu        sync.Mutex
	ctx       context.Context
	cancel    context.CancelFunc
	servers   map[string]*snmpMIBServer
	reloadNow chan struct{}
	wg        sync.WaitGroup
}

func NewSNMPMIBAgentManager(svc *Service, logger *zap.Logger) *SNMPMIBAgentManager {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &SNMPMIBAgentManager{svc: svc, logger: logger.Named("page-config-snmp-mib-agent")}
}

func (m *SNMPMIBAgentManager) Start(ctx context.Context) error {
	if m == nil || m.svc == nil {
		return nil
	}
	m.mu.Lock()
	if m.cancel != nil {
		m.mu.Unlock()
		return nil
	}
	m.ctx, m.cancel = context.WithCancel(ctx)
	m.servers = map[string]*snmpMIBServer{}
	m.reloadNow = make(chan struct{}, 1)
	m.mu.Unlock()

	m.reload(m.ctx)
	m.wg.Add(1)
	go m.loop()
	return nil
}

func (m *SNMPMIBAgentManager) Stop() error {
	if m == nil {
		return nil
	}
	m.mu.Lock()
	if m.cancel != nil {
		m.cancel()
	}
	servers := m.servers
	m.servers = nil
	m.cancel = nil
	m.ctx = nil
	m.mu.Unlock()

	var firstErr error
	for _, server := range servers {
		if err := server.Stop(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	m.wg.Wait()
	return firstErr
}

func (m *SNMPMIBAgentManager) RequestReload() {
	if m == nil {
		return
	}
	m.mu.Lock()
	ch := m.reloadNow
	m.mu.Unlock()
	if ch == nil {
		return
	}
	select {
	case ch <- struct{}{}:
	default:
	}
}

func (m *SNMPMIBAgentManager) loop() {
	defer m.wg.Done()
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		m.mu.Lock()
		ctx := m.ctx
		ch := m.reloadNow
		m.mu.Unlock()
		if ctx == nil {
			return
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.reload(ctx)
		case <-ch:
			m.reload(ctx)
		}
	}
}

func (m *SNMPMIBAgentManager) reload(ctx context.Context) {
	if m == nil || m.svc == nil {
		return
	}
	targets, err := m.svc.listActiveSNMPMIBTargetsForServe(ctx)
	if err != nil {
		m.logger.Warn("list SNMP MIB query targets failed", zap.Error(err))
		return
	}
	desired := desiredSNMPMIBServerConfigs(targets)

	m.mu.Lock()
	defer m.mu.Unlock()
	if m.ctx == nil {
		return
	}
	for key, server := range m.servers {
		next, ok := desired[key]
		if !ok || !server.config.equal(next) {
			if err := server.Stop(); err != nil {
				m.logger.Warn("stop SNMP MIB agent failed", zap.String("addr", key), zap.Error(err))
			}
			delete(m.servers, key)
		}
	}
	for key, config := range desired {
		if _, ok := m.servers[key]; ok {
			continue
		}
		server := newSNMPMIBServer(m.svc, config, m.logger)
		if err := server.Start(m.ctx); err != nil {
			m.logger.Warn("start SNMP MIB agent failed", zap.String("addr", key), zap.Error(err))
			continue
		}
		m.servers[key] = server
		m.logger.Info("SNMP MIB query agent started", zap.String("addr", server.Address()))
	}
}

type snmpMIBServerConfig struct {
	ListenIP    string
	ListenPort  int
	Communities []string
	Users       []snmpMIBUser
}

type snmpMIBUser struct {
	Username       string
	AuthProtocol   string
	AuthCredential string
	PrivProtocol   string
	PrivCredential string
}

func (c snmpMIBServerConfig) address() string {
	return net.JoinHostPort(firstNonEmpty(strings.TrimSpace(c.ListenIP), "0.0.0.0"), strconv.Itoa(c.ListenPort))
}

func (c snmpMIBServerConfig) equal(other snmpMIBServerConfig) bool {
	if c.address() != other.address() || len(c.Communities) != len(other.Communities) || len(c.Users) != len(other.Users) {
		return false
	}
	for i := range c.Communities {
		if c.Communities[i] != other.Communities[i] {
			return false
		}
	}
	for i := range c.Users {
		if c.Users[i] != other.Users[i] {
			return false
		}
	}
	return true
}

func desiredSNMPMIBServerConfigs(targets []SNMPAlarmTarget) map[string]snmpMIBServerConfig {
	grouped := map[string]snmpMIBServerConfig{}
	communitySets := map[string]map[string]struct{}{}
	userSets := map[string]map[string]snmpMIBUser{}
	for _, target := range targets {
		if !target.MIBQueryEnabled {
			continue
		}
		config := snmpMIBServerConfig{ListenIP: target.ListenIP, ListenPort: target.ListenPort}
		key := config.address()
		existing := grouped[key]
		if existing.ListenPort == 0 {
			existing = config
		}
		if strings.EqualFold(target.Version, "v2") {
			community := strings.TrimSpace(target.Community)
			if community == "" {
				continue
			}
			if communitySets[key] == nil {
				communitySets[key] = map[string]struct{}{}
			}
			communitySets[key][community] = struct{}{}
			grouped[key] = existing
			continue
		}
		if strings.EqualFold(target.Version, "v3") {
			user := snmpMIBUser{
				Username:       strings.TrimSpace(target.SecurityName),
				AuthProtocol:   strings.ToUpper(strings.TrimSpace(target.AuthProtocol)),
				AuthCredential: target.AuthCredential,
				PrivProtocol:   strings.ToUpper(strings.TrimSpace(target.PrivProtocol)),
				PrivCredential: target.PrivCredential,
			}
			if user.Username == "" {
				continue
			}
			if userSets[key] == nil {
				userSets[key] = map[string]snmpMIBUser{}
			}
			userSets[key][snmpMIBUserKey(user)] = user
			grouped[key] = existing
		}
	}
	for key, set := range communitySets {
		communities := make([]string, 0, len(set))
		for community := range set {
			communities = append(communities, community)
		}
		sort.Strings(communities)
		config := grouped[key]
		config.Communities = communities
		grouped[key] = config
	}
	for key, set := range userSets {
		users := make([]snmpMIBUser, 0, len(set))
		for _, user := range set {
			users = append(users, user)
		}
		sort.Slice(users, func(i, j int) bool {
			return snmpMIBUserKey(users[i]) < snmpMIBUserKey(users[j])
		})
		config := grouped[key]
		config.Users = users
		grouped[key] = config
	}
	return grouped
}

func snmpMIBUserKey(user snmpMIBUser) string {
	return strings.Join([]string{user.Username, user.AuthProtocol, user.AuthCredential, user.PrivProtocol, user.PrivCredential}, "\x00")
}

type snmpMIBServer struct {
	svc    *Service
	config snmpMIBServerConfig
	logger *zap.Logger

	conn                 *net.UDPConn
	ctx                  context.Context
	cancel               context.CancelFunc
	wg                   sync.WaitGroup
	startedAt            time.Time
	unknownEngineIDCount uint32

	cacheMu     sync.Mutex
	cache       *snmpMIBSnapshot
	cacheExpiry time.Time
}

func newSNMPMIBServer(svc *Service, config snmpMIBServerConfig, logger *zap.Logger) *snmpMIBServer {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &snmpMIBServer{svc: svc, config: config, logger: logger}
}

func (s *snmpMIBServer) Start(ctx context.Context) error {
	if s == nil || s.svc == nil {
		return nil
	}
	if s.svc.alarmStore == nil {
		return fmt.Errorf("SNMP MIB alarm store is not configured")
	}
	addr, err := net.ResolveUDPAddr("udp", s.config.address())
	if err != nil {
		return err
	}
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return err
	}
	s.conn = conn
	s.ctx, s.cancel = context.WithCancel(ctx)
	s.startedAt = time.Now()
	s.wg.Add(1)
	go s.readLoop()
	return nil
}

func (s *snmpMIBServer) Stop() error {
	if s == nil {
		return nil
	}
	if s.cancel != nil {
		s.cancel()
	}
	var err error
	if s.conn != nil {
		err = s.conn.Close()
	}
	s.wg.Wait()
	return err
}

func (s *snmpMIBServer) Address() string {
	if s == nil || s.conn == nil {
		return ""
	}
	return s.conn.LocalAddr().String()
}

func (s *snmpMIBServer) readLoop() {
	defer s.wg.Done()
	buf := make([]byte, snmpMaxOIDResponse)
	for {
		n, remote, err := s.conn.ReadFromUDP(buf)
		if err != nil {
			if s.ctx == nil || s.ctx.Err() != nil || strings.Contains(err.Error(), "use of closed network connection") {
				return
			}
			s.logger.Warn("read SNMP MIB query packet failed", zap.Error(err))
			continue
		}
		data := make([]byte, n)
		copy(data, buf[:n])
		s.wg.Add(1)
		go func() {
			defer s.wg.Done()
			s.handlePacket(data, remote)
		}()
	}
}

func (s *snmpMIBServer) handlePacket(data []byte, remote *net.UDPAddr) {
	packet, err := s.decodePacket(data)
	if err != nil || packet == nil {
		return
	}
	if !s.requestAllowed(packet) {
		return
	}
	if packet.Version == g.Version3 && s.shouldReportV3EngineID(packet) {
		s.sendV3EngineIDReport(packet, remote)
		return
	}
	switch packet.PDUType {
	case g.GetRequest, g.GetNextRequest, g.GetBulkRequest:
	default:
		return
	}
	ctx, cancel := context.WithTimeout(s.ctx, 5*time.Second)
	defer cancel()
	snapshot, err := s.snapshot(ctx)
	if err != nil {
		s.logger.Warn("load SNMP MIB alarm snapshot failed", zap.Error(err))
		return
	}
	response := s.responsePacket(packet, snapshot)
	wire, err := response.MarshalMsg()
	if err != nil {
		s.logger.Warn("marshal SNMP MIB response failed", zap.Error(err))
		return
	}
	if _, err := s.conn.WriteToUDP(wire, remote); err != nil {
		s.logger.Warn("write SNMP MIB response failed", zap.String("remote", remote.String()), zap.Error(err))
	}
}

func (s *snmpMIBServer) decodePacket(data []byte) (*g.SnmpPacket, error) {
	decoder := s.v3Decoder(true)
	packet, err := decoder.UnmarshalTrap(data, true)
	if err == nil || len(s.config.Users) == 0 {
		return packet, err
	}
	return s.v3Decoder(false).UnmarshalTrap(data, true)
}

func (s *snmpMIBServer) requestAllowed(packet *g.SnmpPacket) bool {
	switch packet.Version {
	case g.Version2c:
		return s.communityAllowed(packet.Community)
	case g.Version3:
		return s.v3UserAllowed(packet)
	default:
		return false
	}
}

func (s *snmpMIBServer) v3Decoder(withUserTable bool) *g.GoSNMP {
	decoder := &g.GoSNMP{
		Version:            g.Version3,
		SecurityModel:      g.UserSecurityModel,
		MsgFlags:           g.NoAuthNoPriv,
		SecurityParameters: s.v3BaseSecurityParameters(""),
		Logger:             snmpAgentGosnmpLogger,
	}
	if withUserTable && len(s.config.Users) > 0 {
		table := g.NewSnmpV3SecurityParametersTable(snmpAgentGosnmpLogger)
		for _, user := range s.config.Users {
			if err := table.Add(user.Username, s.v3UserSecurityParameters(user)); err != nil {
				s.logger.Warn("register SNMP v3 MIB user failed", zap.String("user", user.Username), zap.Error(err))
			}
		}
		decoder.TrapSecurityParametersTable = table
	}
	return decoder
}

func (s *snmpMIBServer) v3BaseSecurityParameters(username string) *g.UsmSecurityParameters {
	return &g.UsmSecurityParameters{
		UserName:                 username,
		AuthoritativeEngineID:    snmpAgentEngineID,
		AuthoritativeEngineBoots: 1,
		AuthoritativeEngineTime:  s.v3EngineTime(),
		Logger:                   snmpAgentGosnmpLogger,
	}
}

func (s *snmpMIBServer) v3UserSecurityParameters(user snmpMIBUser) *g.UsmSecurityParameters {
	params := s.v3BaseSecurityParameters(user.Username)
	params.AuthenticationProtocol = snmpAgentAuthProtocol(user.AuthProtocol)
	params.AuthenticationPassphrase = user.AuthCredential
	params.PrivacyProtocol = snmpAgentPrivProtocol(user.PrivProtocol)
	params.PrivacyPassphrase = user.PrivCredential
	return params
}

func (s *snmpMIBServer) v3EngineTime() uint32 {
	if s == nil || s.startedAt.IsZero() {
		return 1
	}
	elapsed := time.Since(s.startedAt) / time.Second
	if elapsed < 1 {
		return 1
	}
	return uint32(elapsed)
}

func (s *snmpMIBServer) v3UserAllowed(packet *g.SnmpPacket) bool {
	if packet == nil {
		return false
	}
	params, _ := packet.SecurityParameters.(*g.UsmSecurityParameters)
	username := ""
	if params != nil {
		username = strings.TrimSpace(params.UserName)
	}
	if username == "" && s.shouldReportV3EngineID(packet) {
		return len(s.config.Users) > 0
	}
	for _, user := range s.config.Users {
		if username == user.Username {
			return true
		}
	}
	return false
}

func (s *snmpMIBServer) shouldReportV3EngineID(packet *g.SnmpPacket) bool {
	if packet == nil || packet.Version != g.Version3 || packet.SecurityModel != g.UserSecurityModel {
		return false
	}
	params, _ := packet.SecurityParameters.(*g.UsmSecurityParameters)
	if params == nil {
		return false
	}
	engineID := params.AuthoritativeEngineID
	if engineID == snmpAgentEngineID {
		return false
	}
	return len(engineID) < 5 || len(engineID) > 32
}

func (s *snmpMIBServer) sendV3EngineIDReport(packet *g.SnmpPacket, remote *net.UDPAddr) {
	if packet == nil || remote == nil {
		return
	}
	params, _ := packet.SecurityParameters.(*g.UsmSecurityParameters)
	if params == nil {
		params = s.v3BaseSecurityParameters("")
	} else {
		params = params.Copy().(*g.UsmSecurityParameters)
		params.AuthoritativeEngineID = snmpAgentEngineID
		params.AuthoritativeEngineBoots = 1
		params.AuthoritativeEngineTime = s.v3EngineTime()
	}
	count := atomic.AddUint32(&s.unknownEngineIDCount, 1)
	packet.PDUType = g.Report
	packet.MsgFlags &= g.AuthPriv
	packet.SecurityParameters = params
	packet.Variables = []g.SnmpPDU{{
		Name:  oidUSMStatsUnknownEngineIDs,
		Type:  g.Counter32,
		Value: count,
	}}
	wire, err := packet.MarshalMsg()
	if err != nil {
		s.logger.Warn("marshal SNMP v3 engineID report failed", zap.Error(err))
		return
	}
	if _, err := s.conn.WriteToUDP(wire, remote); err != nil {
		s.logger.Warn("write SNMP v3 engineID report failed", zap.String("remote", remote.String()), zap.Error(err))
	}
}

func (s *snmpMIBServer) responsePacket(packet *g.SnmpPacket, snapshot *snmpMIBSnapshot) *g.SnmpPacket {
	variables := s.responseVariables(packet, snapshot)
	if packet.Version == g.Version3 {
		packet.PDUType = g.GetResponse
		packet.Error = g.NoError
		packet.ErrorIndex = 0
		packet.Variables = variables
		return packet
	}
	return &g.SnmpPacket{
		Version:   packet.Version,
		Community: packet.Community,
		PDUType:   g.GetResponse,
		RequestID: packet.RequestID,
		Error:     g.NoError,
		Variables: variables,
	}
}

func (s *snmpMIBServer) communityAllowed(community string) bool {
	community = strings.TrimSpace(community)
	for _, allowed := range s.config.Communities {
		if community == allowed {
			return true
		}
	}
	return false
}

func snmpAgentAuthProtocol(protocol string) g.SnmpV3AuthProtocol {
	switch strings.ToUpper(strings.TrimSpace(protocol)) {
	case "MD5":
		return g.MD5
	case "SHA", "SHA1":
		return g.SHA
	case "SHA224":
		return g.SHA224
	case "SHA256":
		return g.SHA256
	case "SHA384":
		return g.SHA384
	case "SHA512":
		return g.SHA512
	default:
		return g.NoAuth
	}
}

func snmpAgentPrivProtocol(protocol string) g.SnmpV3PrivProtocol {
	switch strings.ToUpper(strings.TrimSpace(protocol)) {
	case "DES":
		return g.DES
	case "AES", "AES128":
		return g.AES
	case "AES192":
		return g.AES192
	case "AES256":
		return g.AES256
	default:
		return g.NoPriv
	}
}

func (s *snmpMIBServer) snapshot(ctx context.Context) (*snmpMIBSnapshot, error) {
	now := time.Now()
	s.cacheMu.Lock()
	if s.cache != nil && now.Before(s.cacheExpiry) {
		out := s.cache
		s.cacheMu.Unlock()
		return out, nil
	}
	s.cacheMu.Unlock()

	snapshot, err := buildSNMPMIBSnapshot(ctx, s.svc.alarmStore, snmpAgentMaxRows)
	if err != nil {
		return nil, err
	}
	s.cacheMu.Lock()
	s.cache = snapshot
	s.cacheExpiry = now.Add(snmpAgentCacheTTL)
	s.cacheMu.Unlock()
	return snapshot, nil
}

func (s *snmpMIBServer) responseVariables(packet *g.SnmpPacket, snapshot *snmpMIBSnapshot) []g.SnmpPDU {
	switch packet.PDUType {
	case g.GetRequest:
		out := make([]g.SnmpPDU, 0, len(packet.Variables))
		for _, variable := range packet.Variables {
			oid := normalizeSNMPOID(variable.Name)
			if entry, ok := snapshot.get(oid); ok {
				out = append(out, entry.pdu())
			} else {
				out = append(out, snmpNoSuchInstancePDU(oid))
			}
		}
		return out
	case g.GetBulkRequest:
		return snmpBulkResponseVariables(packet, snapshot)
	default:
		out := make([]g.SnmpPDU, 0, len(packet.Variables))
		for _, variable := range packet.Variables {
			oid := normalizeSNMPOID(variable.Name)
			if entry, ok := snapshot.nextAfter(oid); ok {
				out = append(out, entry.pdu())
			} else {
				out = append(out, snmpEndOfMibViewPDU(oid))
			}
		}
		return out
	}
}

func snmpBulkResponseVariables(packet *g.SnmpPacket, snapshot *snmpMIBSnapshot) []g.SnmpPDU {
	if len(packet.Variables) == 0 {
		return nil
	}
	nonRepeaters := int(packet.NonRepeaters)
	if nonRepeaters > len(packet.Variables) {
		nonRepeaters = len(packet.Variables)
	}
	maxRepetitions := int(packet.MaxRepetitions)
	if maxRepetitions <= 0 {
		maxRepetitions = 1
	}
	if maxRepetitions > snmpMaxBulkRows {
		maxRepetitions = snmpMaxBulkRows
	}
	out := make([]g.SnmpPDU, 0, len(packet.Variables)*maxRepetitions)
	for i := 0; i < nonRepeaters; i++ {
		oid := normalizeSNMPOID(packet.Variables[i].Name)
		if entry, ok := snapshot.nextAfter(oid); ok {
			out = append(out, entry.pdu())
		} else {
			out = append(out, snmpEndOfMibViewPDU(oid))
		}
	}
	for i := nonRepeaters; i < len(packet.Variables); i++ {
		oid := normalizeSNMPOID(packet.Variables[i].Name)
		for rep := 0; rep < maxRepetitions; rep++ {
			entry, ok := snapshot.nextAfter(oid)
			if !ok {
				out = append(out, snmpEndOfMibViewPDU(oid))
				break
			}
			out = append(out, entry.pdu())
			oid = entry.OID
		}
	}
	return out
}

type snmpMIBSnapshot struct {
	entries []snmpMIBEntry
	byOID   map[string]snmpMIBEntry
}

type snmpMIBEntry struct {
	OID   string
	Type  g.Asn1BER
	Value any
}

func (e snmpMIBEntry) pdu() g.SnmpPDU {
	return g.SnmpPDU{Name: e.OID, Type: e.Type, Value: e.Value}
}

func (s *snmpMIBSnapshot) get(oid string) (snmpMIBEntry, bool) {
	if s == nil {
		return snmpMIBEntry{}, false
	}
	entry, ok := s.byOID[normalizeSNMPOID(oid)]
	return entry, ok
}

func (s *snmpMIBSnapshot) nextAfter(oid string) (snmpMIBEntry, bool) {
	if s == nil || len(s.entries) == 0 {
		return snmpMIBEntry{}, false
	}
	oid = normalizeSNMPOID(oid)
	idx := sort.Search(len(s.entries), func(i int) bool {
		return compareSNMPOID(s.entries[i].OID, oid) > 0
	})
	if idx >= len(s.entries) {
		return snmpMIBEntry{}, false
	}
	return s.entries[idx], true
}

func buildSNMPMIBSnapshot(ctx context.Context, store alarm.AlarmStore, limit int) (*snmpMIBSnapshot, error) {
	if store == nil {
		return nil, fmt.Errorf("SNMP MIB alarm store is not configured")
	}
	alarms, err := querySNMPActiveAlarms(ctx, store, limit)
	if err != nil {
		return nil, err
	}
	entries := make([]snmpMIBEntry, 0, len(alarms)*len(defaultSNMPAlarmFields()))
	usedIDs := map[int]struct{}{}
	mapper := nbsnmp.DefaultAlarmMapper()
	for _, item := range alarms {
		notificationID := uniqueSNMPNotificationID(snmpNotificationIDFromAlarm(item), usedIDs)
		snmpAlarm := snmpAlarmFromModel(item, event.SubjectAlarmRaised)
		if snmpAlarm.Extra == nil {
			snmpAlarm.Extra = map[string]string{}
		}
		snmpAlarm.Extra["notificationID"] = strconv.Itoa(notificationID)
		vars, err := mapper.MapAlarmToTrapPDU(snmpAlarm)
		if err != nil {
			continue
		}
		for _, variable := range vars {
			entries = append(entries, snmpMIBEntry{
				OID:   normalizeSNMPOID(variable.OID) + "." + strconv.Itoa(notificationID),
				Type:  snmpVarTypeToBER(variable.Type),
				Value: variable.Value,
			})
		}
	}
	sort.Slice(entries, func(i, j int) bool {
		return compareSNMPOID(entries[i].OID, entries[j].OID) < 0
	})
	byOID := make(map[string]snmpMIBEntry, len(entries))
	for _, entry := range entries {
		byOID[entry.OID] = entry
	}
	return &snmpMIBSnapshot{entries: entries, byOID: byOID}, nil
}

func querySNMPActiveAlarms(ctx context.Context, store alarm.AlarmStore, limit int) ([]model.Alarm, error) {
	if limit <= 0 {
		limit = snmpAgentMaxRows
	}
	pageSize := minPositive(limit, snmpAgentPageSize)
	out := make([]model.Alarm, 0, pageSize)
	for page := 1; len(out) < limit; page++ {
		resp, err := store.ListActive(ctx, alarm.AlarmFilter{
			ListRequest: model.ListRequest{Page: page, PageSize: pageSize},
		})
		if err != nil {
			return nil, fmt.Errorf("query active SNMP MIB alarms: %w", err)
		}
		if resp == nil || len(resp.Items) == 0 {
			break
		}
		out = append(out, resp.Items...)
		if len(resp.Items) < pageSize {
			break
		}
	}
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func uniqueSNMPNotificationID(id int, used map[int]struct{}) int {
	if id <= 0 {
		id = 1
	}
	for attempts := 0; attempts < snmpMaxNotificationID; attempts++ {
		if _, ok := used[id]; !ok {
			used[id] = struct{}{}
			return id
		}
		id++
		if id > snmpMaxNotificationID {
			id = 1
		}
	}
	return 1
}

func snmpVarTypeToBER(t nbsnmp.VarType) g.Asn1BER {
	switch t {
	case nbsnmp.VarTypeInteger:
		return g.Integer
	case nbsnmp.VarTypeCounter32:
		return g.Counter32
	case nbsnmp.VarTypeCounter64:
		return g.Counter64
	case nbsnmp.VarTypeTimeTicks:
		return g.TimeTicks
	case nbsnmp.VarTypeIPAddress:
		return g.IPAddress
	case nbsnmp.VarTypeObjectID:
		return g.ObjectIdentifier
	default:
		return g.OctetString
	}
}

func snmpAlarmEntryRootOID() string {
	return strings.TrimSuffix(OIDNotificationIDValue, ".1")
}

func snmpNoSuchInstancePDU(oid string) g.SnmpPDU {
	return g.SnmpPDU{Name: normalizeSNMPOID(oid), Type: g.NoSuchInstance, Value: nil}
}

func snmpEndOfMibViewPDU(oid string) g.SnmpPDU {
	return g.SnmpPDU{Name: normalizeSNMPOID(oid), Type: g.EndOfMibView, Value: nil}
}

func normalizeSNMPOID(oid string) string {
	return strings.Trim(strings.TrimSpace(oid), ".")
}

func compareSNMPOID(left, right string) int {
	leftParts := parseSNMPOIDParts(left)
	rightParts := parseSNMPOIDParts(right)
	maxLen := len(leftParts)
	if len(rightParts) > maxLen {
		maxLen = len(rightParts)
	}
	for i := 0; i < maxLen; i++ {
		var l, r uint64
		if i < len(leftParts) {
			l = leftParts[i]
		}
		if i < len(rightParts) {
			r = rightParts[i]
		}
		if l < r {
			return -1
		}
		if l > r {
			return 1
		}
	}
	return 0
}

func parseSNMPOIDParts(oid string) []uint64 {
	oid = normalizeSNMPOID(oid)
	if oid == "" {
		return nil
	}
	parts := strings.Split(oid, ".")
	out := make([]uint64, 0, len(parts))
	for _, part := range parts {
		value, err := strconv.ParseUint(part, 10, 64)
		if err != nil {
			return nil
		}
		out = append(out, value)
	}
	return out
}
