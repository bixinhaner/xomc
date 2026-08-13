import { Card, Form, Input, Radio, Space, Typography } from 'antd';
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
const HTTP_PROTOCOL = 'http';
const HTTP_PORT = '8080';
const HTTPS_PROTOCOL = 'https';
const HTTPS_PORT = '8443';
const DEFAULT_UPLOAD_PATH = '/smallcell/FileUploadService';
const DEFAULT_DOWNLOAD_PATH = '/smallcell/FileDownloadService';
const STANDARD_ADDRESS_MAX_WIDTH = 520;

const cardTitleStyle: React.CSSProperties = {
	fontSize: 14,
	fontWeight: 600,
};

const protocolOptionsStyle: React.CSSProperties = {
	width: '100%',
};

const protocolOptionStyle: React.CSSProperties = {
	display: 'flex',
	alignItems: 'flex-start',
	gap: 10,
	width: '100%',
	padding: '10px 12px',
	border: '1px solid var(--color-border)',
	borderRadius: 6,
	background: 'var(--color-neutral-0)',
	cursor: 'pointer',
};

const protocolOptionSelectedStyle: React.CSSProperties = {
	border: '1px solid var(--color-primary-500)',
	background: 'var(--color-primary-50)',
};

const protocolOptionTitleStyle: React.CSSProperties = {
	fontWeight: 600,
};

const protocolOptionDescriptionStyle: React.CSSProperties = {
	display: 'block',
	marginTop: 2,
	fontSize: 12,
	lineHeight: 1.5,
	color: 'var(--color-text-secondary)',
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

async function validateHTTPTransferAddress(
	value: unknown,
	hostError: string,
	baseURLError: string,
	httpSchemeError: string,
): Promise<void> {
	const raw = typeof value === 'string' ? value : '';
	if (!raw) return;
	const result = parseTransferAddress(raw, { protocol: HTTP_PROTOCOL, port: HTTP_PORT });
	if (result.kind === 'standard') return;
	if (/^https:/i.test(raw)) throw new Error(httpSchemeError);
	throw new Error(/^https?:/i.test(raw) ? baseURLError : hostError);
}

function validateHTTPSBaseURL(value: unknown, errorMessage: string): Promise<void> {
	const raw = typeof value === 'string' ? value : '';
	if (!raw) return Promise.resolve();
	const result = parseTransferAddress(raw, { protocol: HTTPS_PROTOCOL, port: HTTPS_PORT });
	return result.kind === 'standard'
		? Promise.resolve()
		: Promise.reject(new Error(errorMessage));
}

interface TransferAddressInputProps extends Omit<
	React.ComponentProps<typeof Input>,
	'value' | 'onChange'
> {
	value?: string;
	onChange?: (value: string) => void;
	protocol?: 'http' | 'https';
	port?: string;
}

function TransferAddressInput({
	value,
	onChange,
	protocol = HTTP_PROTOCOL,
	port = HTTP_PORT,
	...inputProps
}: TransferAddressInputProps) {
	const parsed = parseTransferAddress(value, { protocol, port });
	const raw = typeof value === 'string' ? value : '';
	const displayValue = parsed.mode === 'standard' ? parsed.host : raw;

	return (
		<AddonInput
			{...inputProps}
			addonBefore={`${protocol}://`}
			addonAfter={`:${port}`}
			compactStyle={{ width: '100%', maxWidth: STANDARD_ADDRESS_MAX_WIDTH }}
			value={displayValue}
			onChange={(event) => {
				const input = event.target.value;
				const isFullURL = /^https?:/i.test(input);
				const hasURLSyntax = /[/\\?#@]/.test(input);
				const hasInvalidWhitespace = input !== input.trim();
				const isInvalidColonInput = input.includes(':') && !isValidTransferHost(input);
				onChange?.(
					isFullURL || hasURLSyntax || hasInvalidWhitespace || isInvalidColonInput
						? input
						: buildStandardBaseURL(input, { protocol, port }),
				);
			}}
		/>
	);
}

export default function TransferSettings({ form }: TransferSettingsProps) {
	const t = useT();
	const selectedProtocolPolicy = Form.useWatch('protocolPolicy', form) ?? 'force_http';
	const protocolOptions = [
		{
			value: 'force_http',
			title: t('system.transfer.forceHTTP'),
			description: t('system.transfer.forceHTTPDescription'),
		},
		{
			value: 'prefer_https',
			title: t('system.transfer.preferHTTPS'),
			description: t('system.transfer.preferHTTPSDescription'),
		},
	];

	return (
		<Form
			form={form}
			layout="vertical"
			size="small"
			initialValues={{
				protocolPolicy: 'force_http',
				uploadPath: DEFAULT_UPLOAD_PATH,
				downloadPath: DEFAULT_DOWNLOAD_PATH,
				uploadMaxFileSize: DEFAULT_MAX_FILE_SIZE,
				maxGlobalUpgradeConcurrency: DEFAULT_MAX_GLOBAL_UPGRADE_CONCURRENCY,
			}}
		>
			<Typography.Paragraph type="secondary">
				{t('system.transfer.inheritHint')}
			</Typography.Paragraph>

			<Form.Item
				name="protocolPolicy"
				label={t('system.transfer.protocolPolicy')}
			>
				<Radio.Group style={protocolOptionsStyle}>
					<Space orientation="vertical" size={8} style={protocolOptionsStyle}>
						{protocolOptions.map((option) => (
							<Radio
								key={option.value}
								value={option.value}
								style={{
									...protocolOptionStyle,
									...(selectedProtocolPolicy === option.value ? protocolOptionSelectedStyle : {}),
								}}
							>
								<span>
									<span style={protocolOptionTitleStyle}>{option.title}</span>
									<span style={protocolOptionDescriptionStyle}>{option.description}</span>
								</span>
							</Radio>
						))}
					</Space>
				</Radio.Group>
			</Form.Item>

			<Card
				size="small"
				title={<span style={cardTitleStyle}>{t('system.transfer.uploadSection')}</span>}
				style={{ marginBottom: 16 }}
			>
				<Form.Item
					name="uploadBaseURL"
					label={t('system.transfer.httpUploadBaseURL')}
					extra={t('system.transfer.serverIPHelp')}
					rules={[
						{
							validator: (_, value) => validateHTTPTransferAddress(
								value,
								t('system.transfer.serverIPInvalid'),
								t('system.transfer.baseURLInvalid'),
								t('system.transfer.httpBaseURLInvalid'),
							),
						},
					]}
				>
					<TransferAddressInput placeholder={t('system.transfer.serverIPPlaceholder')} />
				</Form.Item>
				<Form.Item
					name="httpsUploadBaseURL"
					label={t('system.transfer.httpsUploadBaseURL')}
					extra={t('system.transfer.httpsBaseURLHelp')}
					dependencies={['protocolPolicy']}
					rules={[
						({ getFieldValue }) => ({
							required: getFieldValue('protocolPolicy') === 'prefer_https',
							message: t('system.transfer.httpsBaseURLRequired'),
						}),
						{
							validator: (_, value) => validateHTTPSBaseURL(
								value,
								t('system.transfer.httpsBaseURLInvalid'),
							),
						},
					]}
				>
					<TransferAddressInput
						protocol={HTTPS_PROTOCOL}
						port={HTTPS_PORT}
						placeholder={t('system.transfer.serverIPPlaceholder')}
					/>
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
					label={t('system.transfer.httpDownloadBaseURL')}
					extra={t('system.transfer.serverIPHelp')}
					rules={[
						{
							validator: (_, value) => validateHTTPTransferAddress(
								value,
								t('system.transfer.serverIPInvalid'),
								t('system.transfer.baseURLInvalid'),
								t('system.transfer.httpBaseURLInvalid'),
							),
						},
					]}
				>
					<TransferAddressInput placeholder={t('system.transfer.serverIPPlaceholder')} />
				</Form.Item>
				<Form.Item
					name="httpsDownloadBaseURL"
					label={t('system.transfer.httpsDownloadBaseURL')}
					extra={t('system.transfer.httpsBaseURLHelp')}
					dependencies={['protocolPolicy']}
					rules={[
						({ getFieldValue }) => ({
							required: getFieldValue('protocolPolicy') === 'prefer_https',
							message: t('system.transfer.httpsBaseURLRequired'),
						}),
						{
							validator: (_, value) => validateHTTPSBaseURL(
								value,
								t('system.transfer.httpsBaseURLInvalid'),
							),
						},
					]}
				>
					<TransferAddressInput
						protocol={HTTPS_PROTOCOL}
						port={HTTPS_PORT}
						placeholder={t('system.transfer.serverIPPlaceholder')}
					/>
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
