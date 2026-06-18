package quicksettings

import (
	"context"
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/core/appconfig"
	"github.com/omcgo/omcgo/internal/core/dictloader"
)

// LoaderName 是 dictloader.Registry 中的注册名。
const LoaderName = "quick-settings"

// Loader 实现 dictloader.Loader 接口(T-0138)。
//
// 加载语义:
//  1. 扫描 {XMLBaseDir}/{Directory}/*.xml(文件名去 .xml = paramModel name)
//  2. 解析为 Group 列表
//  3. 写入 Registry(按 paramModel name 索引,原子替换)
//
// 不写 PG / 不写 Redis;每 paramModel 数十项常量级数据,进程内 sync.RWMutex 足够。
type Loader struct {
	cfg      appconfig.QuickSettingsLoaderConfig
	base     string
	registry *Registry
	logger   *zap.Logger
}

// NewLoader 构造 Loader。registry 必须非空(由调用方注入,便于路由 handler 共用同一实例)。
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

// Registry 返回 Loader 持有的 Registry,路由 handler 可直接用同一实例。
func (l *Loader) Registry() *Registry { return l.registry }

func (l *Loader) run(_ context.Context) (dictloader.Report, error) {
	rep := dictloader.NewReport(LoaderName)
	defer rep.Finish()

	dir := filepath.Join(l.base, l.cfg.Directory)
	entries, err := os.ReadDir(dir)
	if err != nil {
		rep.AddError(l.cfg.Directory, "read_dir", err)
		// 目录不存在不阻塞启动(quicksettings 是可选功能)
		l.logger.Warn("quicksettings: directory not found, no params loaded",
			zap.String("dir", dir), zap.Error(err))
		return rep, nil
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(name, ".xml") {
			continue
		}
		paramModel := strings.TrimSuffix(name, ".xml")
		rep.FilesScanned++

		path := filepath.Join(dir, name)
		raw, err := os.ReadFile(path)
		if err != nil {
			rep.FilesSkipped++
			rep.AddError(name, "read", err)
			l.logger.Warn("quicksettings: skip file", zap.String("file", path), zap.Error(err))
			continue
		}

		var doc xmlQuickSettings
		if err := xml.Unmarshal(raw, &doc); err != nil {
			rep.AddError(name, "parse", err)
			return rep, fmt.Errorf("xml unmarshal %s: %w", path, err)
		}

		// 文件名 vs XML 顶层 paramModel 属性一致性校验(可选属性,缺省时只用文件名)
		if doc.ParamModel != "" && doc.ParamModel != paramModel {
			err := fmt.Errorf("xml paramModel=%q mismatches file name=%q", doc.ParamModel, paramModel)
			rep.AddError(name, "validate", err)
			return rep, err
		}

		groups, err := buildGroups(doc, name)
		if err != nil {
			rep.AddError(name, "validate", err)
			return rep, fmt.Errorf("validate %s: %w", path, err)
		}

		l.registry.Replace(paramModel, groups)
		rep.FilesLoaded++
		rep.RowsAffected += len(groups)
		l.logger.Info("quicksettings: loaded",
			zap.String("file", name),
			zap.String("paramModel", paramModel),
			zap.Int("groups", len(groups)))
	}

	return rep, nil
}

// xmlQuickSettings 镜像 XML 顶层结构。
type xmlQuickSettings struct {
	XMLName    xml.Name   `xml:"quickSettings"`
	ParamModel string     `xml:"paramModel,attr"` // 可选,缺省时用文件名
	Groups     []xmlGroup `xml:"group"`
}

type xmlGroup struct {
	ID             string     `xml:"id,attr"`
	TitleZh        string     `xml:"titleZh,attr"`
	TitleEn        string     `xml:"titleEn,attr"`
	MultiInstance  string     `xml:"multiInstance,attr"`
	ObjectPath     string     `xml:"objectPath,attr"`
	MaxInstances   int        `xml:"maxInstances,attr"`
	Style          string     `xml:"style,attr"`
	ParentSelector string     `xml:"parentSelector,attr"`
	Params         []xmlParam `xml:"param"`
}

type xmlParam struct {
	Name            string         `xml:"name,attr"`
	TitleZh         string         `xml:"titleZh,attr"`
	TitleEn         string         `xml:"titleEn,attr"`
	StandardPath    string         `xml:"standardPath,attr"`
	Leaf            string         `xml:"leaf,attr"`
	Type            string         `xml:"type,attr"`
	Required        string         `xml:"required,attr"`
	Readonly        string         `xml:"readonly,attr"`
	Hint            string         `xml:"hint,attr"`
	MinValue        string         `xml:"minValue,attr"`
	MaxValue        string         `xml:"maxValue,attr"`
	CheckboxOptions string         `xml:"checkboxOptions,attr"`
	ExtraInfoPath   string         `xml:"extraInfoPath,attr"`
	Unit            string         `xml:"unit,attr"`
	HideRangeHint   string         `xml:"hideRangeHint,attr"`
	EnumOptions    []xmlEnumOption `xml:"option"`
}

type xmlEnumOption struct {
	Value string `xml:"value,attr"`
	Label string `xml:"label,attr"`
}

// buildGroups 把 XML 解码结果转换为领域 Group 列表,并做最小一致性校验。
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
				// 多实例 group: leaf 与 standardPath 至少有一个;若给了 standardPath
				// 但未给 leaf,则按 objectPath 反推 leaf;若同时给了二者,校验一致。
				if p.Leaf == "" && p.StandardPath == "" {
					return nil, fmt.Errorf("%s: group %s param %s missing leaf or standardPath", fileName, g.ID, p.Name)
				}
				if p.StandardPath != "" {
					if !strings.HasPrefix(p.StandardPath, g.ObjectPath) {
						return nil, fmt.Errorf("%s: group %s param %s standardPath %q must start with group objectPath %q",
							fileName, g.ID, p.Name, p.StandardPath, g.ObjectPath)
					}
					derived := strings.TrimPrefix(p.StandardPath, g.ObjectPath)
					if p.Leaf == "" {
						p.Leaf = derived
					} else if p.Leaf != derived {
						return nil, fmt.Errorf("%s: group %s param %s leaf %q inconsistent with standardPath %q (expect %q)",
							fileName, g.ID, p.Name, p.Leaf, p.StandardPath, derived)
					}
				}
			} else {
				if p.StandardPath == "" {
					return nil, fmt.Errorf("%s: group %s param %s missing standardPath", fileName, g.ID, p.Name)
				}
			}
			var enums []EnumOption
			if len(p.EnumOptions) > 0 {
				enums = make([]EnumOption, 0, len(p.EnumOptions))
				for _, eo := range p.EnumOptions {
					if eo.Value == "" {
						return nil, fmt.Errorf("%s: group %s param %s option missing value", fileName, g.ID, p.Name)
					}
					label := eo.Label
					if label == "" {
						label = eo.Value
					}
					enums = append(enums, EnumOption{Value: eo.Value, Label: label})
				}
			}
			var checkboxes []string
			if p.CheckboxOptions != "" {
				for _, s := range strings.Split(p.CheckboxOptions, ",") {
					s = strings.TrimSpace(s)
					if s != "" {
						checkboxes = append(checkboxes, s)
					}
				}
			}
			var minPtr, maxPtr *int64
			if p.MinValue != "" {
				v, err := parseInt64(p.MinValue)
				if err != nil {
					return nil, fmt.Errorf("%s: group %s param %s minValue invalid: %w", fileName, g.ID, p.Name, err)
				}
				minPtr = &v
			}
			if p.MaxValue != "" {
				v, err := parseInt64(p.MaxValue)
				if err != nil {
					return nil, fmt.Errorf("%s: group %s param %s maxValue invalid: %w", fileName, g.ID, p.Name, err)
				}
				maxPtr = &v
			}
			params = append(params, Param{
				Name:            p.Name,
				TitleZh:         p.TitleZh,
				TitleEn:         p.TitleEn,
				StandardPath:    p.StandardPath,
				Leaf:            p.Leaf,
				Type:            p.Type,
				Required:        strings.EqualFold(p.Required, "true"),
				Readonly:        strings.EqualFold(p.Readonly, "true"),
				Hint:            p.Hint,
				MinValue:        minPtr,
				MaxValue:        maxPtr,
				EnumOptions:     enums,
				CheckboxOptions: checkboxes,
				ExtraInfoPath:   strings.TrimSpace(p.ExtraInfoPath),
				Unit:            strings.TrimSpace(p.Unit),
				HideRangeHint:   strings.EqualFold(p.HideRangeHint, "true"),
			})
		}
		out = append(out, Group{
			ID:             g.ID,
			TitleZh:        g.TitleZh,
			TitleEn:        g.TitleEn,
			MultiInstance:  multi,
			ObjectPath:     g.ObjectPath,
			MaxInstances:   g.MaxInstances,
			Style:          g.Style,
			ParentSelector: g.ParentSelector,
			Params:         params,
		})
	}
	return out, nil
}

func parseInt64(s string) (int64, error) {
	return strconv.ParseInt(strings.TrimSpace(s), 10, 64)
}
