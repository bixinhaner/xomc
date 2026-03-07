import type { ReactNode } from 'react';
import { Steps } from 'antd';
import type { StepProps } from 'antd';
import { useThemeToken } from '@/hooks/useThemeToken';

interface WizardStep {
  title: string;
  description?: string;
  status?: StepProps['status'];
}

interface Props {
  /** Step definitions for the left sidebar */
  steps: WizardStep[];
  /** Currently active step index (0-based) */
  currentStep: number;
  /** Main content for the current step */
  children: ReactNode;
  /** Footer content (navigation buttons) */
  footer?: ReactNode;
}

/**
 * WizardPageLayout — vertical step sidebar (200px) + right content area.
 *
 * Used by multi-step configuration wizards.
 */
export default function WizardPageLayout({
  steps,
  currentStep,
  children,
  footer,
}: Props) {
  const token = useThemeToken();
  return (
    <div
      style={{
        display: 'flex',
        width: '100%',
        height: '100%',
        background: token.colorBgContainer,
        borderRadius: 6,
        border: `1px solid ${token.colorBorderSecondary}`,
        overflow: 'hidden',
      }}
    >
      {/* Left step sidebar */}
      <div
        style={{
          width: 200,
          flexShrink: 0,
          padding: '24px 16px',
          borderRight: `1px solid ${token.colorBorderSecondary}`,
          background: token.colorBgLayout,
          display: 'flex',
          flexDirection: 'column',
          gap: 16,
          overflowY: 'auto',
        }}
      >
        <Steps
          direction="vertical"
          current={currentStep}
          size="small"
          items={steps.map((step) => ({
            title: step.title,
            description: step.description,
            status: step.status,
          }))}
          style={{ flex: 1 }}
        />
      </div>

      {/* Right content area */}
      <div
        style={{
          flex: 1,
          minWidth: 0,
          display: 'flex',
          flexDirection: 'column',
          overflow: 'hidden',
        }}
      >
        {/* Step content */}
        <div
          style={{
            flex: 1,
            padding: '24px 32px',
            overflowY: 'auto',
          }}
        >
          {children}
        </div>

        {/* Footer navigation */}
        {footer && (
          <div
            style={{
              padding: '12px 32px',
              borderTop: `1px solid ${token.colorBorderSecondary}`,
              background: token.colorBgLayout,
              display: 'flex',
              justifyContent: 'flex-end',
              gap: 8,
              flexShrink: 0,
            }}
          >
            {footer}
          </div>
        )}
      </div>
    </div>
  );
}
