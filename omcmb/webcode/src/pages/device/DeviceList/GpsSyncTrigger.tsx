import { useState } from 'react';
import { WarningOutlined } from '@ant-design/icons';
import { Button, Tooltip, theme } from 'antd';

interface GpsSyncTriggerProps {
  label: string;
  onClick: () => void;
}

export default function GpsSyncTrigger({ label, onClick }: GpsSyncTriggerProps) {
  const { token } = theme.useToken();
  const [tooltipOpen, setTooltipOpen] = useState(false);

  return (
    <Tooltip title={label} open={tooltipOpen} onOpenChange={setTooltipOpen}>
      <Button
        type="text"
        size="small"
        shape="circle"
        icon={<WarningOutlined />}
        aria-label={label}
        onClick={(event) => {
          event.stopPropagation();
          setTooltipOpen(false);
          onClick();
        }}
        style={{
          width: 24,
          minWidth: 24,
          height: 24,
          padding: 0,
          color: token.colorWarning,
        }}
      />
    </Tooltip>
  );
}
