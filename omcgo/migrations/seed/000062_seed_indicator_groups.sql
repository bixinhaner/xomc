-- +goose Up
-- T-0098 收尾：indicator_group_{enb,gsm,gnb} 功能集树从 seed 加载（设计 §2.2 — 22/8/15 节点）。
-- 数据来源：旧 seed/000027（已删除，git 历史 commit 4eb84dca^）。
-- 与 dictloader.ensureDefaultGroups 协作：dictloader 仅插 1 个 'default' 占位组；
-- seed 提供完整 22+8+15 业务功能集树（ON CONFLICT DO NOTHING 不冲突）。

-- ENB 22 节点
INSERT INTO indicator_group_enb (id, en_name, operator_code, is_build_in, description, parent_id, cn_name) VALUES ('0b360fa301e045659792785174a7b177', 'RLC', NULL, '1', NULL, 'e1e2466f156f44cfa116985008f2f298', 'RLC') ON CONFLICT (id) DO NOTHING;
INSERT INTO indicator_group_enb (id, en_name, operator_code, is_build_in, description, parent_id, cn_name) VALUES ('0d75790f9a8141b5a0b81743b17a9f43', 'EQPT', NULL, '1', NULL, 'e1e2466f156f44cfa116985008f2f298', 'EQPT') ON CONFLICT (id) DO NOTHING;
INSERT INTO indicator_group_enb (id, en_name, operator_code, is_build_in, description, parent_id, cn_name) VALUES ('0eea2c28de9c4dc599b41942b8415982', 'MR', NULL, '1', NULL, 'e1e2466f156f44cfa116985008f2f298', 'MR') ON CONFLICT (id) DO NOTHING;
INSERT INTO indicator_group_enb (id, en_name, operator_code, is_build_in, description, parent_id, cn_name) VALUES ('15c2a1900bdf3744bc7288eace7bce82', 'Call', NULL, '1', NULL, 'e1e2466f156f44cfa116985008f2f298', 'Call') ON CONFLICT (id) DO NOTHING;
INSERT INTO indicator_group_enb (id, en_name, operator_code, is_build_in, description, parent_id, cn_name) VALUES ('1798a814848341638bf02d2684aaaaf8', 'HO', NULL, '1', NULL, 'e1e2466f156f44cfa116985008f2f298', 'HO') ON CONFLICT (id) DO NOTHING;
INSERT INTO indicator_group_enb (id, en_name, operator_code, is_build_in, description, parent_id, cn_name) VALUES ('1fea2c28de9c4dc599b41942b8415993', 'TA', NULL, '1', NULL, 'e1e2466f156f44cfa116985008f2f298', 'TA') ON CONFLICT (id) DO NOTHING;
INSERT INTO indicator_group_enb (id, en_name, operator_code, is_build_in, description, parent_id, cn_name) VALUES ('217b1219e4c045b8902460b77a06d7c5', 'USER', NULL, '1', NULL, 'e1e2466f156f44cfa116985008f2f298', 'USER') ON CONFLICT (id) DO NOTHING;
INSERT INTO indicator_group_enb (id, en_name, operator_code, is_build_in, description, parent_id, cn_name) VALUES ('217b1219e4c045b8902460b77a06d7cd', 'S1SIG', NULL, '1', NULL, 'e1e2466f156f44cfa116985008f2f298', 'S1SIG') ON CONFLICT (id) DO NOTHING;
INSERT INTO indicator_group_enb (id, en_name, operator_code, is_build_in, description, parent_id, cn_name) VALUES ('5013b8c4cad2435d85cc69f0fdea4c86', 'DRB', NULL, '1', NULL, 'e1e2466f156f44cfa116985008f2f298', 'DRB') ON CONFLICT (id) DO NOTHING;
INSERT INTO indicator_group_enb (id, en_name, operator_code, is_build_in, description, parent_id, cn_name) VALUES ('7802acaec16b438cb1fb7bbbcc32fc4d', 'S1', NULL, '1', NULL, 'e1e2466f156f44cfa116985008f2f298', 'S1') ON CONFLICT (id) DO NOTHING;
INSERT INTO indicator_group_enb (id, en_name, operator_code, is_build_in, description, parent_id, cn_name) VALUES ('8281bb78ac5d4741b964255e80eb1475', 'RRU', NULL, '1', NULL, 'e1e2466f156f44cfa116985008f2f298', 'RRU') ON CONFLICT (id) DO NOTHING;
INSERT INTO indicator_group_enb (id, en_name, operator_code, is_build_in, description, parent_id, cn_name) VALUES ('84df5915a23f4455b3c88b6600ef0a41', 'PHY', NULL, '1', NULL, 'e1e2466f156f44cfa116985008f2f298', 'PHY') ON CONFLICT (id) DO NOTHING;
INSERT INTO indicator_group_enb (id, en_name, operator_code, is_build_in, description, parent_id, cn_name) VALUES ('8577fdb10211442db907213a6cfc74d9', 'RRC', NULL, '1', NULL, 'e1e2466f156f44cfa116985008f2f298', 'RRC') ON CONFLICT (id) DO NOTHING;
INSERT INTO indicator_group_enb (id, en_name, operator_code, is_build_in, description, parent_id, cn_name) VALUES ('9b7159356d9a49318da8c9a153d7b1cc', 'PAG', NULL, '1', NULL, 'e1e2466f156f44cfa116985008f2f298', 'PAG') ON CONFLICT (id) DO NOTHING;
INSERT INTO indicator_group_enb (id, en_name, operator_code, is_build_in, description, parent_id, cn_name) VALUES ('bdbc60d0ad04458e869c5958e44c2fb4', 'IRATHO', NULL, '1', NULL, 'e1e2466f156f44cfa116985008f2f298', 'IRATHO') ON CONFLICT (id) DO NOTHING;
INSERT INTO indicator_group_enb (id, en_name, operator_code, is_build_in, description, parent_id, cn_name) VALUES ('c89c8327a43a4f4f9a39a3149e14e64f', 'PDCP', NULL, '1', NULL, 'e1e2466f156f44cfa116985008f2f298', 'PDCP') ON CONFLICT (id) DO NOTHING;
INSERT INTO indicator_group_enb (id, en_name, operator_code, is_build_in, description, parent_id, cn_name) VALUES ('cf4b6a280da96231d91dad98f572b370', 'ENDC MN', NULL, '1', NULL, 'e1e2466f156f44cfa116985008f2f298', 'ENDC MN') ON CONFLICT (id) DO NOTHING;
INSERT INTO indicator_group_enb (id, en_name, operator_code, is_build_in, description, parent_id, cn_name) VALUES ('d26c0cc45b434f83b5408e5c9da0fc43', 'CONTEXT', NULL, '1', NULL, 'e1e2466f156f44cfa116985008f2f298', 'CONTEXT') ON CONFLICT (id) DO NOTHING;
INSERT INTO indicator_group_enb (id, en_name, operator_code, is_build_in, description, parent_id, cn_name) VALUES ('e1e2466f156f44cfa116985008f2f298', 'ENB KPI Function Set', NULL, '1', NULL, 'e1e2466f156f44cfa116985008f2f298', 'ENB指标功能集') ON CONFLICT (id) DO NOTHING;
INSERT INTO indicator_group_enb (id, en_name, operator_code, is_build_in, description, parent_id, cn_name) VALUES ('e67f611fb0a14db388e76987d9d3ccc6', 'Customize', NULL, '1', NULL, 'e1e2466f156f44cfa116985008f2f298', 'Customize') ON CONFLICT (id) DO NOTHING;
INSERT INTO indicator_group_enb (id, en_name, operator_code, is_build_in, description, parent_id, cn_name) VALUES ('e8d5e91c19ab4aabab6fac8cab21ec28', 'MAC', NULL, '1', NULL, 'e1e2466f156f44cfa116985008f2f298', 'MAC') ON CONFLICT (id) DO NOTHING;
INSERT INTO indicator_group_enb (id, en_name, operator_code, is_build_in, description, parent_id, cn_name) VALUES ('f4e7edd6102544ca99c864798ec6328d', 'ERAB', NULL, '1', NULL, 'e1e2466f156f44cfa116985008f2f298', 'ERAB') ON CONFLICT (id) DO NOTHING;

-- GSM 8 节点
INSERT INTO indicator_group_gsm (id, en_name, operator_code, is_build_in, description, parent_id, cn_name) VALUES ('27e19486c0854394edf9d6f369b3e674', 'BTS', NULL, '1', NULL, '5b9df11bf8f711f09ab456b61e61b1f6', 'BTS') ON CONFLICT (id) DO NOTHING;
INSERT INTO indicator_group_gsm (id, en_name, operator_code, is_build_in, description, parent_id, cn_name) VALUES ('5b9df074f8f711f09ab456b61e61b1f6', 'TCH', NULL, '1', NULL, '5b9df11bf8f711f09ab456b61e61b1f6', 'TCH') ON CONFLICT (id) DO NOTHING;
INSERT INTO indicator_group_gsm (id, en_name, operator_code, is_build_in, description, parent_id, cn_name) VALUES ('5b9df079f8f711f09ab456b61e61b1f6', 'Call', NULL, '1', NULL, '5b9df11bf8f711f09ab456b61e61b1f6', 'Call') ON CONFLICT (id) DO NOTHING;
INSERT INTO indicator_group_gsm (id, en_name, operator_code, is_build_in, description, parent_id, cn_name) VALUES ('5b9df07ef8f711f09ab456b61e61b1f6', 'HO', NULL, '1', NULL, '5b9df11bf8f711f09ab456b61e61b1f6', 'HO') ON CONFLICT (id) DO NOTHING;
INSERT INTO indicator_group_gsm (id, en_name, operator_code, is_build_in, description, parent_id, cn_name) VALUES ('5b9df0bff8f711f09ab456b61e61b1f6', 'SDCCH', NULL, '1', NULL, '5b9df11bf8f711f09ab456b61e61b1f6', 'SDCCH') ON CONFLICT (id) DO NOTHING;
INSERT INTO indicator_group_gsm (id, en_name, operator_code, is_build_in, description, parent_id, cn_name) VALUES ('5b9df11bf8f711f09ab456b61e61b1f6', 'GSM KPI Function Set', NULL, '1', NULL, '5b9df11bf8f711f09ab456b61e61b1f6', 'GSM指标功能集') ON CONFLICT (id) DO NOTHING;
INSERT INTO indicator_group_gsm (id, en_name, operator_code, is_build_in, description, parent_id, cn_name) VALUES ('5b9df11ff8f711f09ab456b61e61b1f6', 'Customize', NULL, '1', NULL, '5b9df11bf8f711f09ab456b61e61b1f6', 'Customize') ON CONFLICT (id) DO NOTHING;
INSERT INTO indicator_group_gsm (id, en_name, operator_code, is_build_in, description, parent_id, cn_name) VALUES ('5b9df12bf8f711f09ab456b61e61b1f6', 'Data', NULL, '1', NULL, '5b9df11bf8f711f09ab456b61e61b1f6', 'Data') ON CONFLICT (id) DO NOTHING;

-- GNB 15 节点
INSERT INTO indicator_group_gnb (id, en_name, operator_code, is_build_in, description, parent_id, cn_name) VALUES ('0ea16268b13bd0817d71fd4e7f91d997', 'RRC', NULL, '1', NULL, '1fc9dd06886497f80c022a7e6ebf3034', 'RRC') ON CONFLICT (id) DO NOTHING;
INSERT INTO indicator_group_gnb (id, en_name, operator_code, is_build_in, description, parent_id, cn_name) VALUES ('1fc9dd06886497f80c022a7e6ebf3034', 'KPI Function Set', NULL, '1', NULL, '1fc9dd06886497f80c022a7e6ebf3034', '指标功能集') ON CONFLICT (id) DO NOTHING;
INSERT INTO indicator_group_gnb (id, en_name, operator_code, is_build_in, description, parent_id, cn_name) VALUES ('22b312e32a4c93a6d3185822d7bb7484', 'CONTEXT', NULL, '1', NULL, '1fc9dd06886497f80c022a7e6ebf3034', 'CONTEXT') ON CONFLICT (id) DO NOTHING;
INSERT INTO indicator_group_gnb (id, en_name, operator_code, is_build_in, description, parent_id, cn_name) VALUES ('2d7a9006c2461f5d6a7dd503c9d2068d', 'RLC', NULL, '1', NULL, '1fc9dd06886497f80c022a7e6ebf3034', 'RLC') ON CONFLICT (id) DO NOTHING;
INSERT INTO indicator_group_gnb (id, en_name, operator_code, is_build_in, description, parent_id, cn_name) VALUES ('390f710978aabe22fa70b7ffcc299fb6', 'ENDC MN', NULL, '1', NULL, '1fc9dd06886497f80c022a7e6ebf3034', 'ENDC MN') ON CONFLICT (id) DO NOTHING;
INSERT INTO indicator_group_gnb (id, en_name, operator_code, is_build_in, description, parent_id, cn_name) VALUES ('3aa8d56f20205678977003cf6ab17081', 'Customize', NULL, '1', NULL, '1fc9dd06886497f80c022a7e6ebf3034', 'Customize') ON CONFLICT (id) DO NOTHING;
INSERT INTO indicator_group_gnb (id, en_name, operator_code, is_build_in, description, parent_id, cn_name) VALUES ('3f028a4ac126c171fdf41515762b7756', 'HO', NULL, '1', NULL, '1fc9dd06886497f80c022a7e6ebf3034', 'HO') ON CONFLICT (id) DO NOTHING;
INSERT INTO indicator_group_gnb (id, en_name, operator_code, is_build_in, description, parent_id, cn_name) VALUES ('4816848bc50b6bed570c69ca381dd9cf', 'IRATHO', NULL, '1', NULL, '1fc9dd06886497f80c022a7e6ebf3034', 'IRATHO') ON CONFLICT (id) DO NOTHING;
INSERT INTO indicator_group_gnb (id, en_name, operator_code, is_build_in, description, parent_id, cn_name) VALUES ('63ae1c277703d48b94eddb0dddd33117', 'MAC', NULL, '1', NULL, '1fc9dd06886497f80c022a7e6ebf3034', 'MAC') ON CONFLICT (id) DO NOTHING;
INSERT INTO indicator_group_gnb (id, en_name, operator_code, is_build_in, description, parent_id, cn_name) VALUES ('9595f3ede546643a84bd3ac8281b932c', 'DRB', NULL, '1', NULL, '1fc9dd06886497f80c022a7e6ebf3034', 'DRB') ON CONFLICT (id) DO NOTHING;
INSERT INTO indicator_group_gnb (id, en_name, operator_code, is_build_in, description, parent_id, cn_name) VALUES ('99dcda4480b3371248ef6d0da8b29a7d', 'FLOW', NULL, '1', NULL, '1fc9dd06886497f80c022a7e6ebf3034', 'FLOW') ON CONFLICT (id) DO NOTHING;
INSERT INTO indicator_group_gnb (id, en_name, operator_code, is_build_in, description, parent_id, cn_name) VALUES ('b26faaffdc005a351763f17af16f77e6', 'NGSIG', NULL, '1', NULL, '1fc9dd06886497f80c022a7e6ebf3034', 'NGSIG') ON CONFLICT (id) DO NOTHING;
INSERT INTO indicator_group_gnb (id, en_name, operator_code, is_build_in, description, parent_id, cn_name) VALUES ('b309d7eab02f607b991b1541ff210587', 'PHY', NULL, '1', NULL, '1fc9dd06886497f80c022a7e6ebf3034', 'PHY') ON CONFLICT (id) DO NOTHING;
INSERT INTO indicator_group_gnb (id, en_name, operator_code, is_build_in, description, parent_id, cn_name) VALUES ('f942ce91d529d43aaf1521067a058828', 'RRU', NULL, '1', NULL, '1fc9dd06886497f80c022a7e6ebf3034', 'RRU') ON CONFLICT (id) DO NOTHING;
INSERT INTO indicator_group_gnb (id, en_name, operator_code, is_build_in, description, parent_id, cn_name) VALUES ('fc7dd1f0a7f0389e060188837e4894dd', 'PDCP', NULL, '1', NULL, '1fc9dd06886497f80c022a7e6ebf3034', 'PDCP') ON CONFLICT (id) DO NOTHING;

-- +goose Down
-- 不可回滚：被 perf_indicators_*.group_id FK 引用，DELETE 会破坏关联。
SELECT 1;
