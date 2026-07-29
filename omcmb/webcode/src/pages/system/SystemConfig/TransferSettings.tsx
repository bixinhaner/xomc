import { Card, Form, Input, Space, Typography } from 'antd';
import { AddonInput, AddonInputNumber } from '@/components/common/InputAddon';
import { useT } from '@/hooks/useT';
import {
	buildStandardBaseURL,
	isValidTransferHost,
	parseTransferAddress,
} from './transferAddress';

interface TransferSettingsProps {
	form: ReturnType<typeof Form.useForm>[0];
}

const DEFAULT_MAX_FILE_SIZE = 1073741824; // 1 GiB
// 系统级（跨任务）升级设备并发上限默认值，与后端 software.DefaultGlobalUpgradeConcurrency 一致
const DEFAULT_MAX_GLOBAL_UPGRADE_CONCURRENCY = 100;
const FIXED_PROTOCOL = 'http://';
const FIXED_PORT = '8080';
const DEFAULT_UPLOAD_PATH = '/smallcell/FileUploadService';
const DEFAULT_DOWNLOAD_PATH = '/smallcell/FileDownloadService';
const STANDARD_ADDRESS_MAX_WIDTH = 520;

const cardTitleStyle: React.CSSProperties = {
	fontSize: 14,
	fontWeight: 600,
};

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

function validateTransferAddress(
	value: unknown,
	hostError: string,
	baseURLError: string,
): Promise<void> {
	const raw = typeof value === 'string' ? value : '';
	const result = parseTransferAddress(raw);
	if (result.kind !== 'invalid') return Promise.resolve();
	return Promise.reject(new Error(/^https?:/i.test(raw) ? baseURLError : hostError));
}

interface TransferAddressInputProps extends Omit<
	React.ComponentProps<typeof Input>,
	'value' | 'onChange'
> {
	value?: string;
	onChange?: (value: string) => void;
}

function TransferAddressInput({
	value,
	onChange,
	...inputProps
}: TransferAddressInputProps) {
	const parsed = parseTransferAddress(value);
	// 标准部署只填写 IP；存量或主动粘贴的完整 URL 原样展示，避免把
	// HTTPS、自定义端口、反向代理前缀静默改写为 HTTP:8080。
	if (parsed.mode === 'full') {
		return (
			<Input
				{...inputProps}
				value={parsed.raw}
				onChange={(event) => {
					const input = event.target.value;
					onChange?.(!input || isValidTransferHost(input) ? buildStandardBaseURL(input) : input);
				}}
			/>
		);
	}

	return (
		<AddonInput
			{...inputProps}
			addonBefore={FIXED_PROTOCOL}
			addonAfter={`:${FIXED_PORT}`}
			compactStyle={{ width: '100%', maxWidth: STANDARD_ADDRESS_MAX_WIDTH }}
			value={parsed.host}
			onChange={(event) => {
				const input = event.target.value;
				const isFullURL = /^https?:/i.test(input);
				const hasURLSyntax = /[/\\?#@]/.test(input);
				const hasInvalidWhitespace = input !== input.trim();
				const isInvalidColonInput = input.includes(':') && !isValidTransferHost(input);
				onChange?.(
					isFullURL || hasURLSyntax || hasInvalidWhitespace || isInvalidColonInput
						? input
						: buildStandardBaseURL(input),
				);
			}}
		/>
	);
}

export default function TransferSettings({ form }: TransferSettingsProps) {
	const t = useT();

	return (
		<Form
			form={form}
			layout="vertical"
			size="small"
			initialValues={{
				uploadPath: DEFAULT_UPLOAD_PATH,
				downloadPath: DEFAULT_DOWNLOAD_PATH,
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
					label={t('system.transfer.uploadServerIP')}
					extra={t('system.transfer.serverIPHelp')}
					rules={[
						{
							validator: (_, value) => validateTransferAddress(
								value,
								t('system.transfer.serverIPInvalid'),
								t('system.transfer.baseURLInvalid'),
							),
						},
					]}
				>
					<TransferAddressInput placeholder={t('system.transfer.serverIPPlaceholder')} />
				</Form.Item>
				<Space orientation="vertical" style={{ width: '100%' }} size={12}>
					<Form.Item
						name="uploadPath"
						label={t('system.transfer.uploadPath')}
						extra={t('system.transfer.uploadPathHelp')}
						rules={[{
							validator: (_, value) => {
								if (!value) return Promise.resolve();
								return isValidServicePath(value as string)
									? Promise.resolve()
									: Promise.reject(new Error(t('system.transfer.pathInvalid')));
							},
						}]}
					>
						<Input placeholder={DEFAULT_UPLOAD_PATH} />
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

			<Card
				size="small"
				title={<span style={cardTitleStyle}>{t('system.transfer.downloadSection')}</span>}
			>
				<Form.Item
					name="downloadBaseURL"
					label={t('system.transfer.downloadServerIP')}
					extra={t('system.transfer.serverIPHelp')}
					rules={[
						{
							validator: (_, value) => validateTransferAddress(
								value,
								t('system.transfer.serverIPInvalid'),
								t('system.transfer.baseURLInvalid'),
							),
						},
					]}
				>
					<TransferAddressInput placeholder={t('system.transfer.serverIPPlaceholder')} />
				</Form.Item>
				<Form.Item
					name="downloadPath"
					label={t('system.transfer.downloadPath')}
					extra={t('system.transfer.downloadPathHelp')}
					rules={[{
						validator: (_, value) => {
							if (!value) return Promise.resolve();
							return isValidServicePath(value as string)
								? Promise.resolve()
								: Promise.reject(new Error(t('system.transfer.pathInvalid')));
						},
					}]}
				>
					<Input placeholder={DEFAULT_DOWNLOAD_PATH} />
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
