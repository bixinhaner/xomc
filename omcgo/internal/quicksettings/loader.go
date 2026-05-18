package quicksettings

import (
	"context"
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"

	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/core/appconfig"
	"github.com/omcgo/omcgo/internal/core/dictloader"
)

// LoaderName 是 dictloader.Registry 中的注册名。
const LoaderName = "quick-settings"

// fileTechMap 把 XML 文件名映射到制式。新增制式时在此扩展。
var fileTechMap = map[string]TechCode{
	"enb.xml": TechLTE,
	"gnb.xml": TechNR,
}

// Loader 实现 dictloader.Loader 接口（T-0138）。
//
// 加载语义：
//  1. 扫描 {XMLBaseDir}/{Directory}/ 下白名单文件（enb.xml / gnb.xml）
//  2. 解析为 Group 列表
//  3. 写入 Registry（按制式索引，原子替换）
//
// 不写 PG / 不写 Redis；55 项常量级数据，进程内 sync.RWMutex 足够。
type Loader struct {
	cfg      appconfig.QuickSettingsLoaderConfig
	base     string
	registry *Registry
	logger   *zap.Logger
}

// NewLoader 构造 Loader。registry 必须非空（由调用方注入，便于路由 handler 共用同一实例）。
func NewLoader(cfg appconfig.QuickSettingsLoaderConfig, baseDir string, registry *Registry, logger *zap.Logger) *Loader {
	if cfg.Directory == "" {
		cfg.Directory = "quicksettings"
	}
	if logger == nil {
		logger = zap.NewNop()
	}
	if registry == nil {
		registry = NewRegistry()
	}
	return &Loader{cfg: cfg, base: baseDir, registry: registry, logger: logger.Named(LoaderName)}
}

func (l *Loader) Name() string      { return LoaderName }
func (l *Loader) Directory() string { return l.cfg.Directory }

func (l *Loader) LoadOnce(ctx context.Context) (dictloader.Report, error) { return l.run(ctx) }
func (l *Loader) Reload(ctx context.Context) (dictloader.Report, error)   { return l.run(ctx) }

// Registry 返回 Loader 持有的 Registry，路由 handler 可直接用同一实例。
func (l *Loader) Registry() *Registry { return l.registry }

func (l *Loader) run(_ context.Context) (dictloader.Report, error) {
	rep := dictloader.NewReport(LoaderName)
	defer rep.Finish()

	dir := filepath.Join(l.base, l.cfg.Directory)
	for fileName, tech := range fileTechMap {
		rep.FilesScanned++
		path := filepath.Join(dir, fileName)
		raw, err := os.ReadFile(path)
		if err != nil {
			rep.FilesSkipped++
			rep.AddError(fileName, "read", err)
			l.logger.Warn("quicksettings: skip file", zap.String("file", path), zap.Error(err))
			continue
		}

		var doc xmlQuickSettings
		if err := xml.Unmarshal(raw, &doc); err != nil {
			rep.AddError(fileName, "parse", err)
			return rep, fmt.Errorf("xml unmarshal %s: %w", path, err)
		}

		groups, err := buildGroups(doc, fileName)
		if err != nil {
			rep.AddError(fileName, "validate", err)
			return rep, fmt.Errorf("validate %s: %w", path, err)
		}

		l.registry.Replace(tech, groups)
		rep.FilesLoaded++
		rep.RowsAffected += len(groups)
		l.logger.Info("quicksettings: loaded",
			zap.String("file", fileName),
			zap.String("tech", string(tech)),
			zap.Int("groups", len(groups)))
	}

	return rep, nil
}

// xmlQuickSettings 镜像 XML 顶层结构。
type xmlQuickSettings struct {
	XMLName xml.Name   `xml:"quickSettings"`
	Tech    string     `xml:"tech,attr"`
	Groups  []xmlGroup `xml:"group"`
}

type xmlGroup struct {
	ID            string     `xml:"id,attr"`
	TitleZh       string     `xml:"titleZh,attr"`
	TitleEn       string     `xml:"titleEn,attr"`
	MultiInstance string     `xml:"multiInstance,attr"`
	ObjectPath    string     `xml:"objectPath,attr"`
	Params        []xmlParam `xml:"param"`
}

type xmlParam struct {
	Name         string `xml:"name,attr"`
	TitleZh      string `xml:"titleZh,attr"`
	TitleEn      string `xml:"titleEn,attr"`
	StandardPath string `xml:"standardPath,attr"`
	Leaf         string `xml:"leaf,attr"`
}

// buildGroups 把 XML 解码结果转换为领域 Group 列表，并做最小一致性校验。
func buildGroups(doc xmlQuickSettings, fileName string) ([]Group, error) {
	out := make([]Group, 0, len(doc.Groups))
	for _, g := range doc.Groups {
		if g.ID == "" {
			return nil, fmt.Errorf("%s: group missing id", fileName)
		}
		multi := g.MultiInstance == "true"
		if multi && g.ObjectPath == "" {
			return nil, fmt.Errorf("%s: group %s is multiInstance but objectPath is empty", fileName, g.ID)
		}
		params := make([]Param, 0, len(g.Params))
		for _, p := range g.Params {
			if p.Name == "" {
				return nil, fmt.Errorf("%s: group %s has param missing name", fileName, g.ID)
			}
			if multi {
				if p.Leaf == "" {
					return nil, fmt.Errorf("%s: group %s param %s missing leaf", fileName, g.ID, p.Name)
				}
			} else {
				if p.StandardPath == "" {
					return nil, fmt.Errorf("%s: group %s param %s missing standardPath", fileName, g.ID, p.Name)
				}
			}
			params = append(params, Param{
				Name:         p.Name,
				TitleZh:      p.TitleZh,
				TitleEn:      p.TitleEn,
				StandardPath: p.StandardPath,
				Leaf:         p.Leaf,
			})
		}
		out = append(out, Group{
			ID:            g.ID,
			TitleZh:       g.TitleZh,
			TitleEn:       g.TitleEn,
			MultiInstance: multi,
			ObjectPath:    g.ObjectPath,
			Params:        params,
		})
	}
	return out, nil
}
