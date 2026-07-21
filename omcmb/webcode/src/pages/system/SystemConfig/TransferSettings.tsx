import { Card, Form, Input, Space, Typography } from 'antd';
import { AddonInputNumber } from '@/components/common/InputAddon';
import { useT } from '@/hooks/useT';

interface TransferSettingsProps {
	form: ReturnType<typeof Form.useForm>[0];
}

const DEFAULT_MAX_FILE_SIZE = 1073741824; // 1 GiB
// 系统级（跨任务）升级设备并发上限默认值，与后端 software.DefaultGlobalUpgradeConcurrency 一致
const DEFAULT_MAX_GLOBAL_UPGRADE_CONCURRENCY = 100;
const BASE_URL_PLACEHOLDER = 'https://acs.example.com:7557';
const UPLOAD_PATH_PLACEHOLDER = '/smallcell/FileUploadService';
const DOWNLOAD_PATH_PLACEHOLDER = '/smallcell/FileDownloadService';

const cardTitleStyle: React.CSSProperties = {
	fontSize: 14,
	fontWeight: 600,
};

function isValidHTTPURL(value: string): boolean {
	try {
		if (value !== value.trim()) return false;
		const url = new URL(value);
		return (
			(url.protocol === 'http:' || url.protocol === 'https:')
			&& Boolean(url.hostname)
			&& !url.username
			&& !url.password
			&& !url.search
			&& !url.hash
		);
	} catch {
		return false;
	}
}

function isValidServicePath(value: string): boolean {
	if (!value || value !== value.trim() || !value.startsWith('/') || value.startsWith('//')) return false;
	if (value.includes('\\') || value.includes('?') || value.includes('#')) return false;
	try {
		return value.split('/').every((segment) => {
			const decoded = decodeURIComponent(segment);
			return decoded !== '.' && decoded !== '..';
		});
	} catch {
		return false;
	}
}

export default function TransferSettings({ form }: TransferSettingsProps) {
	const t = useT();

	return (
		<Form
			form={form}
			layout="vertical"
			size="small"
			initialValues={{
				uploadMaxFileSize: DEFAULT_MAX_FILE_SIZE,
				maxGlobalUpgradeConcurrency: DEFAULT_MAX_GLOBAL_UPGRADE_CONCURRENCY,
			}}
		>
			<Typography.Paragraph type="secondary">
				{t('system.transfer.inheritHint')} {t('system.transfer.deviceReachabilityHelp')}
			</Typography.Paragraph>

			<Card
				size="small"
				title={<span style={cardTitleStyle}>{t('system.transfer.uploadSection')}</span>}
				style={{ marginBottom: 16 }}
			>
				<Form.Item
					name="uploadBaseURL"
					label={t('system.transfer.uploadBaseURL')}
					extra={t('system.transfer.baseURLHelp')}
					rules={[
						{
							validator: (_, value) => {
								if (!value) return Promise.resolve();
								return isValidHTTPURL(value as string)
									? Promise.resolve()
									: Promise.reject(new Error(t('system.transfer.baseURLInvalid')));
							},
						},
					]}
				>
					<Input placeholder={BASE_URL_PLACEHOLDER} />
				</Form.Item>
				<Space orientation="vertical" style={{ width: '100%' }} size={12}>
					<Form.Item
						name="uploadPath"
						label={t('system.transfer.uploadPath')}
						extra={t('system.transfer.pathHelp')}
						rules={[{
							validator: (_, value) => {
								if (!value) return Promise.resolve();
								return isValidServicePath(value as string)
									? Promise.resolve()
									: Promise.reject(new Error(t('system.transfer.pathInvalid')));
							},
						}]}
					>
						<Input placeholder={UPLOAD_PATH_PLACEHOLDER} />
					</Form.Item>
					<Form.Item
						name="uploadMaxFileSize"
						label={t('system.transfer.uploadMaxFileSize')}
						extra={t('system.transfer.uploadMaxFileSizeHelp')}
					>
						{/* antd6 addonAfter 已废弃：受控 AddonInputNumber 复刻 "bytes" 后缀盒子。 */}
						<AddonInputNumber addonAfter="bytes" min={0} precision={0} style={{ width: '100%' }} />
					</Form.Item>
				</Space>
			</Card>

			{/* 下发服务：路径写死 */}
			<Card
				size="small"
				title={<span style={cardTitleStyle}>{t('system.transfer.downloadSection')}</span>}
			>
				<Form.Item
					name="downloadBaseURL"
					label={t('system.transfer.downloadBaseURL')}
					extra={t('system.transfer.baseURLHelp')}
					rules={[
						{
							validator: (_, value) => {
								if (!value) return Promise.resolve();
								return isValidHTTPURL(value as string)
									? Promise.resolve()
									: Promise.reject(new Error(t('system.transfer.baseURLInvalid')));
							},
						},
					]}
				>
					<Input placeholder={BASE_URL_PLACEHOLDER} />
				</Form.Item>
				<Form.Item
					name="downloadPath"
					label={t('system.transfer.downloadPath')}
					extra={t('system.transfer.pathHelp')}
					rules={[{
						validator: (_, value) => {
							if (!value) return Promise.resolve();
							return isValidServicePath(value as string)
								? Promise.resolve()
								: Promise.reject(new Error(t('system.transfer.pathInvalid')));
						},
					}]}
				>
					<Input placeholder={DOWNLOAD_PATH_PLACEHOLDER} />
				</Form.Item>
				<Form.Item
					name="maxGlobalUpgradeConcurrency"
					label={t('system.transfer.maxGlobalUpgradeConcurrency')}
					extra={t('system.transfer.maxGlobalUpgradeConcurrencyHelp')}
					rules={[{ required: true, message: t('system.transfer.maxGlobalUpgradeConcurrencyRequired') }]}
				>
					<AddonInputNumber addonAfter={t('software.upgrade.units')} min={1} max={10000} precision={0} style={{ width: '100%' }} />
				</Form.Item>
			</Card>

			<Typography.Paragraph type="secondary" style={{ marginTop: 12, marginBottom: 0 }}>
				{t('system.transfer.runtimeHint')}
			</Typography.Paragraph>
		</Form>
	);
}
