import { useState, useCallback, useMemo } from 'react';
import { Input, Modal, Tag, Typography } from 'antd';
import { CheckCircleOutlined, CloseCircleOutlined } from '@ant-design/icons';
import { useT } from '@/hooks/useT';

const { TextArea } = Input;

interface BatchSnModalProps {
  open: boolean;
  onClose: () => void;
  onConfirm: (sns: string[]) => void;
  existingSns: Set<string>; // 已选设备的SN
  allDeviceSns: Set<string>; // 所有可用设备的SN
}

export default function BatchSnModal({
  open,
  onClose,
  onConfirm,
  existingSns,
  allDeviceSns,
}: BatchSnModalProps) {
  const t = useT();
  const [inputValue, setInputValue] = useState('');
  const [result, setResult] = useState<{
    success: string[];
    failed: string[];
    duplicate: string[];
  } | null>(null);

  // 解析输入的SN列表
  const parsedSns = useMemo(() => {
    return inputValue
      .split(/[\n,;]+/)
      .map((s) => s.trim().toUpperCase())
      .filter(Boolean);
  }, [inputValue]);

  // 确认添加
  const handleConfirm = useCallback(() => {
    if (parsedSns.length === 0) {
      return;
    }

    // 分类处理
    const success: string[] = [];
    const failed: string[] = [];
    const duplicate: string[] = [];

    parsedSns.forEach((sn) => {
      if (existingSns.has(sn)) {
        duplicate.push(sn);
      } else if (allDeviceSns.has(sn)) {
        success.push(sn);
      } else {
        failed.push(sn);
      }
    });

    // 显示结果
    setResult({ success, failed, duplicate });
  }, [parsedSns, existingSns, allDeviceSns]);

  // 完成添加
  const handleFinish = useCallback(() => {
    if (result && result.success.length > 0) {
      onConfirm(result.success);
    }
    setInputValue('');
    setResult(null);
    onClose();
  }, [result, onConfirm, onClose]);

  // 继续添加
  const handleContinue = useCallback(() => {
    setResult(null);
  }, []);

  // 关闭弹窗
  const handleClose = useCallback(() => {
    setInputValue('');
    setResult(null);
    onClose();
  }, [onClose]);

  // 是否显示结果
  const showResult = result !== null;

  return (
    <Modal
      title={showResult ? t('mml.console.batchAddResult') : '批量输入设备SN'}
      open={open}
      onOk={showResult ? handleFinish : handleConfirm}
      onCancel={showResult ? handleContinue : handleClose}
      okText={showResult ? t('common.finish') : t('common.confirm')}
      cancelText={showResult ? t('mml.console.continueAdd') : t('common.cancel')}
      width={480}
      destroyOnClose
    >
      {showResult ? (
        // 显示结果
        <div>
          {/* 成功 */}
          {result!.success.length > 0 && (
            <div style={{ marginBottom: 16 }}>
              <div style={{ marginBottom: 8 }}>
                <CheckCircleOutlined style={{ color: '#52c41a', marginRight: 6 }} />
                <Typography.Text strong style={{ color: '#52c41a' }}>
                  {t('mml.console.addSuccessCount', { count: result!.success.length })}
                </Typography.Text>
              </div>
              <div style={{ maxHeight: 100, overflow: 'auto' }}>
                {result!.success.map((sn) => (
                  <Tag key={sn} color="success" style={{ margin: '2px 4px' }}>
                    {sn}
                  </Tag>
                ))}
              </div>
            </div>
          )}

          {/* 重复 */}
          {result!.duplicate.length > 0 && (
            <div style={{ marginBottom: 16 }}>
              <div style={{ marginBottom: 8 }}>
                <Typography.Text type="warning" strong>
                  {t('mml.console.alreadyExistsSkipped', { count: result!.duplicate.length })}
                </Typography.Text>
              </div>
              <div style={{ maxHeight: 80, overflow: 'auto' }}>
                {result!.duplicate.map((sn) => (
                  <Tag key={sn} color="warning" style={{ margin: '2px 4px' }}>
                    {sn}
                  </Tag>
                ))}
              </div>
            </div>
          )}

          {/* 失败 */}
          {result!.failed.length > 0 && (
            <div>
              <div style={{ marginBottom: 8 }}>
                <CloseCircleOutlined style={{ color: '#ff4d4f', marginRight: 6 }} />
                <Typography.Text type="danger" strong>
                  {t('mml.console.deviceNotFound', { count: result!.failed.length })}
                </Typography.Text>
              </div>
              <div style={{ maxHeight: 100, overflow: 'auto' }}>
                {result!.failed.map((sn) => (
                  <Tag key={sn} color="error" style={{ margin: '2px 4px' }}>
                    {sn}
                  </Tag>
                ))}
              </div>
            </div>
          )}

          {result!.success.length === 0 && result!.failed.length === 0 && result!.duplicate.length === 0 && (
            <Typography.Text type="secondary">{t('mml.console.noValidDeviceSN')}</Typography.Text>
          )}
        </div>
      ) : (
        // 输入界面
        <div>
          <div style={{ marginBottom: 12 }}>
            <Typography.Text type="secondary" style={{ fontSize: 12 }}>
              {t('mml.console.snInputHint')}
            </Typography.Text>
          </div>
          <TextArea
            rows={8}
            value={inputValue}
            onChange={(e) => setInputValue(e.target.value)}
            placeholder={'ENB00001\nENB00002\nENB00003'}
            style={{ fontFamily: 'monospace', fontSize: 12 }}
          />
          <div style={{ marginTop: 8, display: 'flex', justifyContent: 'space-between' }}>
            <Typography.Text type="secondary" style={{ fontSize: 11 }}>
              {t('mml.console.selectedDeviceCount', { count: existingSns.size })}
            </Typography.Text>
            <Typography.Text type="secondary" style={{ fontSize: 11 }}>
              {t('mml.console.pendingParseCount', { count: parsedSns.length })}
            </Typography.Text>
          </div>
        </div>
      )}
    </Modal>
  );
}
