import type { ReactNode } from 'react';
import { Button } from 'antd';
import type { ButtonProps } from 'antd';
import { PlusOutlined } from '@ant-design/icons';

interface AddRowButtonProps extends Omit<ButtonProps, 'block' | 'icon' | 'type'> {
  icon?: ReactNode;
}

export function AddRowButton({
  icon = <PlusOutlined />,
  ...props
}: AddRowButtonProps) {
  return (
    <Button
      {...props}
      type="dashed"
      block
      icon={icon}
    />
  );
}
