// Package dictsource 数据字典数据源白名单 + 同步引擎(T-0182)。
//
// 数据流:
//
//	YAML(sources.yaml) → Registry → Resolve(table/field) → SyncEngine 拼 SQL → sys_dictionary_details
//
// 安全:任何不在白名单的表名/字段名永远拼不进 SQL —— 杜绝 SQL 注入。
// 拒绝标准:Resolve 返 (zero, false),调用方需拒绝请求。
package dictsource

import (
	_ "embed"
	"fmt"

	"gopkg.in/yaml.v3"
)

//go:embed sources.yaml
var defaultYAML []byte

// FieldSpec 描述源表中一个可作为字典 label/value 的字段。
type FieldSpec struct {
	Column      string `yaml:"column"`
	DisplayName string `yaml:"display"`
	Type        string `yaml:"type"`
}

// TableSpec 描述一张可作为字典数据源的白名单表。
type TableSpec struct {
	Table       string      `yaml:"table"`
	DisplayName string      `yaml:"display_name"`
	Fields      []FieldSpec `yaml:"fields"`
}

// fileShape 是 sources.yaml 的根节点。
type fileShape struct {
	Sources []TableSpec `yaml:"sources"`
}

// Registry 是只读的白名单查询表。
// 进程启动时 LoadDefault / LoadFromBytes 一次,后续多协程并发只读。
type Registry struct {
	tables       []TableSpec
	tableIndex   map[string]int            // table → tables[i]
	fieldIndex   map[string]map[string]int // table → field → tables[i].Fields[j]
}

// LoadDefault 加载内置白名单(embed sources.yaml)。
// 生产 / 测试默认走这条路径;Registry 进程内单例。
func LoadDefault() (*Registry, error) {
	return LoadFromBytes(defaultYAML)
}

// LoadFromBytes 从任意 YAML bytes 构造 Registry。
// 测试可注入自定义白名单(覆盖更窄/更宽场景)。
func LoadFromBytes(data []byte) (*Registry, error) {
	var f fileShape
	if err := yaml.Unmarshal(data, &f); err != nil {
		return nil, fmt.Errorf("unmarshal dict sources yaml: %w", err)
	}
	if len(f.Sources) == 0 {
		return nil, fmt.Errorf("dict sources yaml: empty sources list")
	}

	reg := &Registry{
		tables:     f.Sources,
		tableIndex: make(map[string]int, len(f.Sources)),
		fieldIndex: make(map[string]map[string]int, len(f.Sources)),
	}
	for i, t := range f.Sources {
		if t.Table == "" {
			return nil, fmt.Errorf("dict sources yaml: entry %d has empty table name", i)
		}
		if _, dup := reg.tableIndex[t.Table]; dup {
			return nil, fmt.Errorf("dict sources yaml: duplicate table %q", t.Table)
		}
		if len(t.Fields) == 0 {
			return nil, fmt.Errorf("dict sources yaml: table %q has no fields", t.Table)
		}
		reg.tableIndex[t.Table] = i
		reg.fieldIndex[t.Table] = make(map[string]int, len(t.Fields))
		for j, fd := range t.Fields {
			if fd.Column == "" {
				return nil, fmt.Errorf("dict sources yaml: table %q field %d has empty column", t.Table, j)
			}
			if _, dup := reg.fieldIndex[t.Table][fd.Column]; dup {
				return nil, fmt.Errorf("dict sources yaml: table %q duplicate field %q", t.Table, fd.Column)
			}
			reg.fieldIndex[t.Table][fd.Column] = j
		}
	}
	return reg, nil
}

// ListTables 返所有白名单表(只读快照,调用方不应修改)。
// 用于 GET /admin/sysDictionary/sources 端点。
func (r *Registry) ListTables() []TableSpec {
	out := make([]TableSpec, len(r.tables))
	copy(out, r.tables)
	return out
}

// Resolve 检查 table 是否在白名单内。
// 命中返回 (spec, true);未命中返回 (zero, false)。
func (r *Registry) Resolve(table string) (TableSpec, bool) {
	idx, ok := r.tableIndex[table]
	if !ok {
		return TableSpec{}, false
	}
	return r.tables[idx], true
}

// ResolveField 检查 (table, column) 组合是否在白名单内。
// 命中返回 (spec, true);未命中(table 不在白名单 / column 不在 table 字段表)返回 (zero, false)。
func (r *Registry) ResolveField(table, column string) (FieldSpec, bool) {
	cols, ok := r.fieldIndex[table]
	if !ok {
		return FieldSpec{}, false
	}
	idx, ok := cols[column]
	if !ok {
		return FieldSpec{}, false
	}
	return r.tables[r.tableIndex[table]].Fields[idx], true
}
