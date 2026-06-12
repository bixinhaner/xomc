-- +goose Up
-- issue #115 调整3：自定义命令 PATH 管理对齐内置命令（方案 A）。
--
-- 精瘦关联表：自定义命令(mml_custom_command) ↔ 标准路径(standard_params)。
-- 设计要点（用户决策）：
--   · 仅存「关联 + 排序 + 默认勾选」；path 元数据(类型/access/描述)与「是否支持」
--     均读时 JOIN standard_params / param_mappings，不在此重复落库。
--   · standard_path_id NOT NULL + FK → path 仅来自字典（不允许任意裸路径）。
--   · ON DELETE：父命令删除级联清理；引用的 standard_params 受 RESTRICT 保护，防误删字典。
--
-- 注意：本迁移仅建表（A1 后端基座）。现网 param_paths(JSONB) 的 backfill 与
-- param_paths↔本表的同步触发器属后续切片 A2，故此处不动 param_paths、不建触发器。
CREATE TABLE IF NOT EXISTS public.mml_custom_command_paths (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    command_id uuid NOT NULL,
    standard_path_id uuid NOT NULL,
    default_selected boolean DEFAULT true NOT NULL,
    sort_order integer DEFAULT 0 NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT mml_custom_command_paths_pkey PRIMARY KEY (id),
    CONSTRAINT fk_ccp_command FOREIGN KEY (command_id)
        REFERENCES public.mml_custom_command(id) ON DELETE CASCADE,
    CONSTRAINT fk_ccp_standard_path FOREIGN KEY (standard_path_id)
        REFERENCES public.standard_params(id) ON DELETE RESTRICT,
    CONSTRAINT uq_ccp_command_path UNIQUE (command_id, standard_path_id)
);

CREATE INDEX IF NOT EXISTS idx_ccp_command_sort
    ON public.mml_custom_command_paths (command_id, sort_order);

COMMENT ON TABLE public.mml_custom_command_paths IS
    'issue #115 调整3：自定义命令↔标准路径精瘦关联表。仅存关联+排序+默认勾选；元数据与是否支持读时 JOIN standard_params/param_mappings，不重复落库。standard_path_id NOT NULL = path 仅来自字典。';

-- +goose Down
DROP TABLE IF EXISTS public.mml_custom_command_paths;
