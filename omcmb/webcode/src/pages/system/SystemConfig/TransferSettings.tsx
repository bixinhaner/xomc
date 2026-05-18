import { Alert, Card, Form, Input, InputNumber, Space, Typography } from 'antd';
import { useT } from '@/hooks/useT';

interface TransferSettingsProps {
	form: ReturnType<typeof Form.useForm>[0];
}

const cardTitleStyle: React.CSSProperties = {
	fontSize: 14,
	fontWeight: 600,
};

const rowStyle: React.CSSProperties = {
	marginBottom: 16,
};

export default function TransferSettings({ form }: TransferSettingsProps) {
	const t = useT();

	return (
		<Form
			form={form}
			layout="vertical"
			size="small"
			initialValues={{
				uploadBaseURL: '',
				uploadPath: '',
				uploadMaxFileSize: undefined,
				downloadBaseURL: '',
				downloadPath: '',
			}}
		>
			<Alert
				showIcon
				type="info"
				style={{ marginBottom: 16 }}
				message={t('system.transfer.inheritHint')}
			/>

			<Card
				size="small"
				title={<span style={cardTitleStyle}>{t('system.transfer.uploadSection')}</span>}
				style={{ marginBottom: 16 }}
			>
				<div style={rowStyle}>
					<Space direction="vertical" style={{ width: '100%' }} size={12}>
						<Form.Item
							name="uploadBaseURL"
							label={t('system.transfer.uploadBaseURL')}
							extra={t('system.transfer.baseURLHelp')}
						>
							<Input placeholder="http://acs.example.com:7557" />
						</Form.Item>
						<Form.Item
							name="uploadPath"
							label={t('system.transfer.uploadPath')}
							extra={t('system.transfer.pathHelp')}
						>
							<Input placeholder="/smallcell/FileUploadService" />
						</Form.Item>
						<Form.Item
							name="uploadMaxFileSize"
							label={t('system.transfer.uploadMaxFileSize')}
							extra={t('system.transfer.uploadMaxFileSizeHelp')}
						>
							<InputNumber min={0} precision={0} style={{ width: '100%' }} placeholder={16777216} />
						</Form.Item>
					</Space>
				</div>
			</Card>

			<Card
				size="small"
				title={<span style={cardTitleStyle}>{t('system.transfer.downloadSection')}</span>}
			>
				<div style={rowStyle}>
					<Space direction="vertical" style={{ width: '100%' }} size={12}>
						<Form.Item
							name="downloadBaseURL"
							label={t('system.transfer.downloadBaseURL')}
							extra={t('system.transfer.baseURLHelp')}
						>
							<Input placeholder="http://acs.example.com:7557" />
						</Form.Item>
						<Form.Item
							name="downloadPath"
							label={t('system.transfer.downloadPath')}
							extra={t('system.transfer.pathHelp')}
						>
							<Input placeholder="/smallcell/FileDownloadService" />
						</Form.Item>
					</Space>
				</div>
			</Card>

			<Typography.Paragraph type="secondary" style={{ marginTop: 12, marginBottom: 0 }}>
				{t('system.transfer.runtimeHint')}
			</Typography.Paragraph>
		</Form>
	);
}