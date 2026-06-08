-- +goose Up
-- T-MMLSRC: mml_commands 按 command_name 去重(command_name 不允许重复)。
-- 背景:同一逻辑命令派生出两个 command_code(短码/空码 vs 带命名空间前缀的限定码),
--       共用同一 command_name。canonical 规则 = 每个重复名保留 command_code 最长的一条
--       (更规范/带前缀/非空),其余命令的 sub_fields 合并到保留命令后删除。
-- 数据特性(实测):每对的 PATH 集合互为包含(无互相独有),故合并无损。
-- sub_fields.command_id 改指保留命令时,AFTER UPDATE 触发器自动重算 target_paths。
-- 幂等:去重后无重复名,重跑为 no-op。fresh 部署(init_seed 含原始重复数据)与存量库都会收敛。
-- +goose StatementBegin
DO $$
DECLARE
    rec     RECORD;
    keep_id uuid;
BEGIN
    FOR rec IN
        SELECT command_name FROM mml_commands GROUP BY command_name HAVING count(*) > 1
    LOOP
        -- canonical:command_code 最长者胜(并列按 command_code、id 取稳定值)
        SELECT id INTO keep_id
          FROM mml_commands
         WHERE command_name = rec.command_name
         ORDER BY length(command_code) DESC, command_code ASC, id ASC
         LIMIT 1;

        -- 合并:把冗余命令的 sub_fields 改指到保留命令;若保留命令已有同一 standard_path
        -- (撞 (command_id, standard_path_id) 唯一键)则跳过(随冗余命令一并 CASCADE 删除)。
        UPDATE mml_command_sub_fields sf
           SET command_id = keep_id
         WHERE sf.command_id IN (
                   SELECT id FROM mml_commands
                    WHERE command_name = rec.command_name AND id <> keep_id)
           AND NOT EXISTS (
                   SELECT 1 FROM mml_command_sub_fields k
                    WHERE k.command_id = keep_id
                      AND k.standard_path_id = sf.standard_path_id);

        -- 删冗余命令(ON DELETE CASCADE 清掉其残留的重复 sub_fields;触发器重算 target_paths)
        DELETE FROM mml_commands
         WHERE command_name = rec.command_name AND id <> keep_id;
    END LOOP;

    -- 合并后 PATH 排查:sub_fields 指向不存在 standard_params 的孤儿 path,记日志供运维核对。
    FOR rec IN
        SELECT c.command_code, c.command_name, sf.standard_path_id
          FROM mml_command_sub_fields sf
          JOIN mml_commands c ON c.id = sf.command_id
          LEFT JOIN standard_params sp ON sp.id = sf.standard_path_id
         WHERE sp.id IS NULL
    LOOP
        RAISE WARNING 'T-MMLSRC orphan path after dedup: command=% (%) standard_path_id=%',
            rec.command_name, rec.command_code, rec.standard_path_id;
    END LOOP;
END $$;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- 去重不可逆(冗余命令已删除);down 无操作。
SELECT 1;
-- +goose StatementEnd
