import React from 'react';
import { CheckOutlined } from '@ant-design/icons';
import { useThemeToken } from '@/hooks/useThemeToken';

export interface WizardStep {
  title: string;
  description?: string;
}

export interface StepWizardProps {
  steps: WizardStep[];
  currentStep: number;
  onStepChange?: (step: number) => void;
  children?: React.ReactNode;
}

const StepWizard: React.FC<StepWizardProps> = ({
  steps,
  currentStep,
  onStepChange,
  children,
}) => {
  const token = useThemeToken();

  return (
    <div style={{ display: 'flex', height: '100%', minHeight: 400 }}>
      {/* Left sidebar */}
      <div
        style={{
          width: 200,
          flexShrink: 0,
          background: token.colorBgLayout,
          borderRight: `1px solid ${token.colorBorderSecondary}`,
          padding: '16px 0',
          display: 'flex',
          flexDirection: 'column',
          gap: 0,
        }}
      >
        {steps.map((step, index) => {
          const isCompleted = index < currentStep;
          const isActive = index === currentStep;
          const isClickable = Boolean(onStepChange) && (isCompleted || isActive);

          return (
            <div
              key={index}
              onClick={() => {
                if (isClickable) onStepChange?.(index);
              }}
              style={{
                display: 'flex',
                alignItems: 'flex-start',
                gap: 10,
                padding: '10px 16px',
                cursor: isClickable ? 'pointer' : 'default',
                background: isActive ? '#e6f7ff' : 'transparent',
                borderLeft: isActive ? `3px solid ${token.colorPrimary}` : '3px solid transparent',
                transition: 'background 0.2s',
              }}
            >
              {/* Step number/check */}
              <div
                style={{
                  width: 24,
                  height: 24,
                  borderRadius: '50%',
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                  flexShrink: 0,
                  fontSize: isCompleted ? 12 : 13,
                  fontWeight: 600,
                  background: isCompleted
                    ? '#52C41A'
                    : isActive
                      ? token.colorPrimary
                      : token.colorBorderSecondary,
                  color: '#fff',
                  marginTop: 1,
                }}
              >
                {isCompleted ? <CheckOutlined style={{ fontSize: 12 }} /> : index + 1}
              </div>

              {/* Step info */}
              <div style={{ flex: 1, minWidth: 0 }}>
                <div
                  style={{
                    fontSize: 13,
                    fontWeight: isActive ? 600 : 400,
                    color: isActive
                      ? token.colorPrimary
                      : isCompleted
                        ? '#52C41A'
                        : token.colorText,
                    lineHeight: 1.4,
                  }}
                >
                  {step.title}
                </div>
                {step.description && (
                  <div
                    style={{
                      fontSize: 12,
                      color: token.colorTextSecondary,
                      marginTop: 2,
                      lineHeight: 1.4,
                      overflow: 'hidden',
                      textOverflow: 'ellipsis',
                      whiteSpace: 'nowrap',
                    }}
                  >
                    {step.description}
                  </div>
                )}
              </div>
            </div>
          );
        })}
      </div>

      {/* Content area */}
      <div style={{ flex: 1, minWidth: 0, overflow: 'auto', padding: 24 }}>
        {children}
      </div>
    </div>
  );
};

export default StepWizard;
