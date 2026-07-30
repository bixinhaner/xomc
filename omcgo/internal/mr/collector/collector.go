package collector

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/omcgo/omcgo/internal/core/compress"
	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/core/model"
	coremodel "github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/core/rawarchive"
	"github.com/omcgo/omcgo/internal/mr"
	"github.com/omcgo/omcgo/internal/mr/parser"
	"go.uber.org/zap"
)

// MRFilePayload is the event payload for MR file received events.
type MRFilePayload struct {
	MinioPath string `json:"minio_path"`
	Bucket    string `json:"bucket"`
	DeviceID  string `json:"device_id"`
	DeviceSN  string `json:"device_sn"`
	Carrier   string `json:"carrier"`
	FileName  string `json:"file_name"`
	FileSize  int64  `json:"file_size"`
}

// DeviceLookup 抽象按 SN 查设备的能力。device.DeviceRepository 满足。
// 用于 payload.DeviceID 为空时（如 acs/upload 直传路径）由 collector 自己查。
type DeviceLookup interface {
	GetBySerialNumber(ctx context.Context, sn string) (*coremodel.Device, error)
}

type rawArchiver interface {
	Schedule(bucket, object string, onTerminal func(context.Context, string, string, string))
	RemoveOld(ctx context.Context, bucket, object string)
}

// MRCollector handles MR file download, type detection, parsing, and storage.
type MRCollector struct {
	minioClient *minio.Client
	bucket      string
	parsers     map[string]parser.MRParser
	store       mr.MRStore
	devices     DeviceLookup // 可 nil；nil 时强制要求 payload.DeviceID 非空
	eventBus    event.EventBus
	archiver    rawArchiver
	logger      *zap.Logger
}

// NewMRCollector creates a new MR file collector.
func NewMRCollector(
	minioClient *minio.Client,
	bucket string,
	store mr.MRStore,
	eventBus event.EventBus,
	logger *zap.Logger,
) *MRCollector {
	parsers := map[string]parser.MRParser{
		MRTypeMRO: parser.NewMROParser(),
		MRTypeMRS: parser.NewMRSParser(),
		MRTypeMRE: parser.NewMREParser(),
	}
	return &MRCollector{
		minioClient: minioClient,
		bucket:      bucket,
		parsers:     parsers,
		store:       store,
		eventBus:    eventBus,
		logger:      logger,
	}
}

// SetDeviceLookup 注入设备查询能力（可选）。注入后 payload.DeviceID 为空时
// collector 会按 device_sn 反查 device_id / carrier。
// 不注入则 payload.DeviceID 必须非空（保留旧 transfer/bridge 路径行为）。
func (c *MRCollector) SetDeviceLookup(d DeviceLookup) {
	c.devices = d
}

// SetArchiver 注入原始文件压缩回写器（issue #836）：入库成功后对明文 MR XML 尝试一次 gzip 回写
// MinIO 省盘（已是 gzip 的真机文件零成本跳过）。Nil-safe — 未注入时不做压缩回写。
func (c *MRCollector) SetArchiver(a *rawarchive.Archiver) {
	c.archiver = a
}

// markRawCompressed 把已压缩回写的 MR 原始对象在 mr_files 标记 raw_compressed=true 并把 minio_path
// 更新为压缩后的新键（issue #836 + 改键 .xml→.xml.gz）。作为 archiver.Schedule 的 onTerminal
// 回调，在压缩 goroutine 内调用；nil-safe。DB 更新成功后删旧明文键（仅改键时）；失败只 warn。
func (c *MRCollector) markRawCompressed(ctx context.Context, bucket, oldObject, newObject string) {
	if c.store == nil {
		return
	}
	if err := c.store.MarkCompressed(ctx, map[string]string{oldObject: newObject}); err != nil {
		c.logger.Warn("mark mr_files raw_compressed", zap.String("object", oldObject), zap.Error(err))
		return
	}
	if oldObject != newObject {
		// 用 archiver 实际压缩的 bucket 删旧键（MR 的 bucket=payload.Bucket 未必等于 c.bucket）。
		c.archiver.RemoveOld(ctx, bucket, oldObject)
	}
}

// SetMRTypeSupport 注入 carrier MR-type 支持判定（可选，#17）。注入后 MRE parser
// 的 "某运营商是否采集 MRE" 决策由 Carrier 适配器（经 CarrierRegistry）给出，
// 取代旧的 "if carrier == cucc" 硬编码。不注入则 MREParser 走内置回退表
// （行为等价，仅 CUCC 不支持 MRE）。
func (c *MRCollector) SetMRTypeSupport(s parser.MRTypeSupportChecker) {
	if s == nil {
		return
	}
	c.parsers[MRTypeMRE] = parser.NewMREParser(s)
}

// Subscribe registers the collector for MR file received events.
func (c *MRCollector) Subscribe(eventBus event.EventBus) error {
	_, err := eventBus.QueueSubscribe(
		event.SubjectMRFileReceived,
		"mr-workers",
		c.handleFileReceived,
	)
	if err != nil {
		return fmt.Errorf("subscribe mr.file.received: %w", err)
	}
	c.logger.Info("MR collector subscribed", zap.String("subject", event.SubjectMRFileReceived))
	return nil
}

func (c *MRCollector) handleFileReceived(ctx context.Context, evt event.Event) error {
	var payload MRFilePayload
	if err := evt.DecodePayload(&payload); err != nil {
		c.logger.Error("decode MR file payload", zap.Error(err))
		return err
	}

	c.logger.Info("processing MR file",
		zap.String("file", payload.FileName),
		zap.String("device_sn", payload.DeviceSN))

	// Detect MR type
	mrType, err := DetectMRType(payload.FileName)
	if err != nil {
		c.logger.Error("detect MR type", zap.Error(err), zap.String("file", payload.FileName))
		return err
	}

	var deviceID uuid.UUID
	carrier := payload.Carrier
	if payload.DeviceID != "" {
		var perr error
		deviceID, perr = uuid.Parse(payload.DeviceID)
		if perr != nil {
			return fmt.Errorf("parse device_id: %w", perr)
		}
	} else if c.devices != nil && payload.DeviceSN != "" {
		// upload handler 直传路径：payload 不带 device_id，按 SN 反查
		dev, derr := c.devices.GetBySerialNumber(ctx, payload.DeviceSN)
		if derr != nil {
			return fmt.Errorf("lookup device by sn %s: %w", payload.DeviceSN, derr)
		}
		if dev == nil {
			c.logger.Warn("MR file uploaded but device not found; skipping",
				zap.String("device_sn", payload.DeviceSN),
				zap.String("file", payload.FileName))
			return nil
		}
		deviceID = dev.ID
		if carrier == "" {
			carrier = string(dev.Carrier)
		}
	} else {
		return fmt.Errorf("mr file payload missing device_id and no DeviceLookup wired")
	}

	// Save file metadata
	fileID := uuid.New()
	now := time.Now()
	bucket := payload.Bucket
	if bucket == "" {
		bucket = c.bucket
	}

	fileInfo := &mr.MRFileInfo{
		ID:          fileID,
		DeviceID:    deviceID,
		DeviceSN:    payload.DeviceSN,
		Carrier:     carrier,
		MRType:      mrType,
		FileName:    payload.FileName,
		FileSize:    payload.FileSize,
		CollectTime: now,
		MinioPath:   payload.MinioPath,
		CreatedAt:   now,
	}

	if err := c.store.SaveFile(ctx, fileInfo); err != nil {
		return fmt.Errorf("save MR file info: %w", err)
	}

	// Download from MinIO
	obj, err := c.minioClient.GetObject(ctx, bucket, payload.MinioPath, minio.GetObjectOptions{})
	if err != nil {
		return fmt.Errorf("get MinIO object %s: %w", payload.MinioPath, err)
	}
	defer obj.Close()

	// #168：单文件体积上限，防超大/异常文件单次全量入内存 OOM。Stat 给确切大小，超限直接拒。
	if stat, statErr := obj.Stat(); statErr == nil {
		if err := ensureMRFileSize(stat.Size); err != nil {
			c.logger.Warn("MR file rejected: oversized",
				zap.String("file", payload.FileName),
				zap.Int64("size", stat.Size),
				zap.Int64("limit", maxMRFileBytes))
			return err
		}
	}

	// Parse
	p, ok := c.parsers[mrType]
	if !ok {
		return fmt.Errorf("no parser for MR type: %s", mrType)
	}

	carrierCode := model.CarrierCode(carrier)
	// issue #321：真机按 TR-069 上传 .xml.gz，MinIO 原样存压缩字节；解析前按 gzip
	// 魔数嗅探透明解压（明文原样透传）。解压在 LimitReader 之前 → 体积上限作用于
	// 解压后内容，兼防 gzip 炸弹。
	decoded, rawCompressed, derr := compress.MaybeGunzip(obj)
	if derr != nil {
		c.logger.Warn("decompress MR file",
			zap.Error(derr),
			zap.String("file", payload.FileName))
		return fmt.Errorf("decompress MR file %s: %w", payload.FileName, derr)
	}
	// io.LimitReader 兜底：Stat 不可用/谎报时,解析最多读 maxMRFileBytes,截断 → 解析报错被捕获。
	data, err := p.Parse(io.LimitReader(decoded, maxMRFileBytes), carrierCode)
	if err != nil {
		c.logger.Warn("parse MR file",
			zap.Error(err),
			zap.String("file", payload.FileName),
			zap.String("type", mrType))
		return err
	}

	// Store parsed records
	if len(data.Records) > 0 {
		if err := c.store.BatchInsertRecords(ctx, fileID, deviceID, mrType, data.Records); err != nil {
			return fmt.Errorf("batch insert MR records: %w", err)
		}
	}

	// Update file as parsed
	if err := c.store.UpdateFileParsed(ctx, fileID, len(data.Records)); err != nil {
		c.logger.Warn("update file parsed status", zap.Error(err))
	}

	// issue #321：入库成功后把原始 MR XML 压缩回写 MinIO 省盘（已 gzip 则零成本跳过）。
	// 异步有界并发，不阻塞 ack；nil-safe。压成功后经 onTerminal 标记 mr_files.raw_compressed=true。
	if err := c.finalizeRawArchive(ctx, bucket, payload.MinioPath, rawCompressed); err != nil {
		return err
	}

	// Publish parsed event
	parsedPayload := map[string]interface{}{
		"file_id":      fileID.String(),
		"device_id":    payload.DeviceID,
		"device_sn":    payload.DeviceSN,
		"mr_type":      mrType,
		"record_count": len(data.Records),
		"file_name":    filepath.Base(payload.FileName),
	}
	parsedEvt, err := event.NewEvent(event.SubjectMRFileParsed, parsedPayload)
	if err == nil {
		if pubErr := c.eventBus.Publish(ctx, event.SubjectMRFileParsed, parsedEvt); pubErr != nil {
			c.logger.Warn("publish mr.file.parsed", zap.Error(pubErr))
		}
	}

	c.logger.Info("MR file processed",
		zap.String("file_id", fileID.String()),
		zap.String("type", mrType),
		zap.Int("records", len(data.Records)))

	return nil
}

// finalizeRawArchive 利用解析阶段已经完成的 gzip 魔数探测，避免对设备原生 gzip
// 对象再次排队、HEAD 和读取。明文对象仍交给异步 archiver 做一次性压缩回写。
func (c *MRCollector) finalizeRawArchive(ctx context.Context, bucket, object string, rawCompressed bool) error {
	if rawCompressed {
		if c.store == nil {
			return nil
		}
		if err := c.store.MarkCompressed(ctx, map[string]string{object: object}); err != nil {
			// 文件记录和解析结果已经成功入库；压缩状态只是存储优化元数据，
			// 失败时不能让整条 NATS 消息重试并重复创建 mr_files 记录。
			c.logger.Warn("mark detected gzip MR object",
				zap.String("object", object), zap.Error(err))
		}
		return nil
	}
	if c.archiver != nil {
		c.archiver.Schedule(bucket, object, c.markRawCompressed)
	}
	return nil
}
