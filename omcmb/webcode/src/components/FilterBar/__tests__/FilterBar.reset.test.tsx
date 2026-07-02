import { beforeEach, describe, expect, it, vi } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { useState } from 'react'
import { IntlProvider } from 'react-intl'
import { zhCN } from '@core/i18n'

import FilterBar, { type FilterField } from '../index'

const eventTypeField: FilterField = {
  name: 'eventType',
  label: '事件类型',
  type: 'select',
  width: 240,
  options: [
    { label: '通信告警', value: 'communication' },
    { label: '服务质量告警', value: 'qualityOfService' },
    { label: '处理出错告警', value: 'processingError' },
    { label: '设备告警', value: 'device' },
  ],
}

function wrap(node: React.ReactElement) {
  return (
    <IntlProvider locale="zh-CN" defaultLocale="zh-CN" messages={zhCN}>
      {node}
    </IntlProvider>
  )
}

function Harness() {
  const [resetTick, setResetTick] = useState(0)

  return (
    <div style={{ width: 720 }}>
      <span data-testid="reset-tick">{resetTick}</span>
      <FilterBar
        filterId="alarm-reset-regression"
        fields={[eventTypeField]}
        initialValues={{ eventType: 'device' }}
        onSearch={vi.fn()}
        onReset={() => setResetTick((value) => value + 1)}
      />
    </div>
  )
}

describe('FilterBar reset', () => {
  beforeEach(() => {
    sessionStorage.clear()
  })

  it('重置后不会把初值重新灌回表单', async () => {
    const user = userEvent.setup()
    render(wrap(<Harness />))

    expect(await screen.findByText('设备告警')).toBeInTheDocument()

    await user.click(screen.getByText('重置'))

    await waitFor(() => {
      expect(screen.queryByText('设备告警')).not.toBeInTheDocument()
    })
    expect(screen.getByTestId('reset-tick')).toHaveTextContent('1')
  })
})