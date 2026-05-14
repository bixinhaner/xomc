import { Steps } from 'antd';
import { useT } from '@/hooks/useT';

export interface StepBarProps {
  current: 1 | 2 | 3 | 4;
  labels?: {
    step1: string;
    step2: string;
    step3: string;
    step4: string;
  };
}

export default function StepBar({ current, labels }: StepBarProps) {
  const t = useT();

  const items = [
    { title: labels?.step1 ?? t('mml.console.stepBar.step1') },
    { title: labels?.step2 ?? t('mml.console.stepBar.step2') },
    { title: labels?.step3 ?? t('mml.console.stepBar.step3') },
    { title: labels?.step4 ?? t('mml.console.stepBar.step4') },
  ];

  return <Steps current={current - 1} items={items} size="small" />;
}
