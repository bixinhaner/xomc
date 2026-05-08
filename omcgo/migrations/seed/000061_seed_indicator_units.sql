-- +goose Up
-- T-0098 收尾：indicator_unit 字典从 seed 加载（设计 §2.3.1 — 27 行完整单位字典）。
-- 数据来源：旧 seed/000027（已删除，git 历史 commit 4eb84dca^）。
-- 与 dictloader.upsertUnits 协作：seed 先插 27（ON CONFLICT DO NOTHING），
-- dictloader 后从 indicator XML 收集 distinct unitId 再 upsert，已有不覆盖。

INSERT INTO indicator_unit (id, en_name, cn_name) VALUES ('%', '%', '百分比') ON CONFLICT (id) DO NOTHING;
INSERT INTO indicator_unit (id, en_name, cn_name) VALUES ('bit', 'bit', '位') ON CONFLICT (id) DO NOTHING;
INSERT INTO indicator_unit (id, en_name, cn_name) VALUES ('Byte', 'Byte', '字节') ON CONFLICT (id) DO NOTHING;
INSERT INTO indicator_unit (id, en_name, cn_name) VALUES ('Byte/s', 'Byte/s', '字节/秒') ON CONFLICT (id) DO NOTHING;
INSERT INTO indicator_unit (id, en_name, cn_name) VALUES ('char', 'char', '字符串') ON CONFLICT (id) DO NOTHING;
INSERT INTO indicator_unit (id, en_name, cn_name) VALUES ('dBm', 'dBm', 'dBm') ON CONFLICT (id) DO NOTHING;
INSERT INTO indicator_unit (id, en_name, cn_name) VALUES ('Erl', 'Erl', 'Erl') ON CONFLICT (id) DO NOTHING;
INSERT INTO indicator_unit (id, en_name, cn_name) VALUES ('Gbit', 'Gbit', '千兆位') ON CONFLICT (id) DO NOTHING;
INSERT INTO indicator_unit (id, en_name, cn_name) VALUES ('GByte', 'GByte', '千兆字节') ON CONFLICT (id) DO NOTHING;
INSERT INTO indicator_unit (id, en_name, cn_name) VALUES ('Kb/PRB', 'Kb/PRB', 'Kb/PRB') ON CONFLICT (id) DO NOTHING;
INSERT INTO indicator_unit (id, en_name, cn_name) VALUES ('Kbit', 'Kbit', '千位') ON CONFLICT (id) DO NOTHING;
INSERT INTO indicator_unit (id, en_name, cn_name) VALUES ('Kbps', 'Kbps', '千位/秒') ON CONFLICT (id) DO NOTHING;
INSERT INTO indicator_unit (id, en_name, cn_name) VALUES ('KByte', 'KByte', '千字节') ON CONFLICT (id) DO NOTHING;
INSERT INTO indicator_unit (id, en_name, cn_name) VALUES ('KByte/s', 'KByte/s', '千字节/秒') ON CONFLICT (id) DO NOTHING;
INSERT INTO indicator_unit (id, en_name, cn_name) VALUES ('Mbps', 'Mbps', '兆位/秒') ON CONFLICT (id) DO NOTHING;
INSERT INTO indicator_unit (id, en_name, cn_name) VALUES ('MByte', 'MByte', '兆字节') ON CONFLICT (id) DO NOTHING;
INSERT INTO indicator_unit (id, en_name, cn_name) VALUES ('milliseconds', 'milliseconds', '毫秒') ON CONFLICT (id) DO NOTHING;
INSERT INTO indicator_unit (id, en_name, cn_name) VALUES ('ms', 'ms', '毫秒') ON CONFLICT (id) DO NOTHING;
INSERT INTO indicator_unit (id, en_name, cn_name) VALUES ('mW', 'mW', '毫瓦') ON CONFLICT (id) DO NOTHING;
INSERT INTO indicator_unit (id, en_name, cn_name) VALUES ('no', 'no', '无') ON CONFLICT (id) DO NOTHING;
INSERT INTO indicator_unit (id, en_name, cn_name) VALUES ('number', 'number', '个') ON CONFLICT (id) DO NOTHING;
INSERT INTO indicator_unit (id, en_name, cn_name) VALUES ('ppm', 'ppm', '百万分比') ON CONFLICT (id) DO NOTHING;
INSERT INTO indicator_unit (id, en_name, cn_name) VALUES ('pps', 'pps', '速率/秒') ON CONFLICT (id) DO NOTHING;
INSERT INTO indicator_unit (id, en_name, cn_name) VALUES ('s', 's', '秒') ON CONFLICT (id) DO NOTHING;
INSERT INTO indicator_unit (id, en_name, cn_name) VALUES ('seconds', 'seconds', '秒') ON CONFLICT (id) DO NOTHING;
INSERT INTO indicator_unit (id, en_name, cn_name) VALUES ('time', 'time', '次数') ON CONFLICT (id) DO NOTHING;
INSERT INTO indicator_unit (id, en_name, cn_name) VALUES ('W', 'W', '瓦特') ON CONFLICT (id) DO NOTHING;

-- +goose Down
-- 不可回滚：dictloader 后续仍会维护被引用的 21 个 unit；强行 DELETE 会破坏外键。
SELECT 1;
