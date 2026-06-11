import { useEffect } from 'react';
import { Card, Form, Input, Space, Typography } from 'antd';
import { AddonInput, AddonInputNumber } from '@/components/common/InputAddon';
import { useT } from '@/hooks/useT';

interface TransferSettingsProps {
	form: ReturnType<typeof Form.useForm>[0];
}

// —— 写死常量（与 ACS YAML / internal/acs/transfercfg 默认保持一致）——
// 设备文件传输目前仅支持 HTTP（nginx 网关无 SSL block），协议与端口固定；
// 用户只需填写网管服务器 IP，前端拼成 http://{ip}:8080 写入上传/下发 BaseURL。
const FIXED_PROTOCOL = 'http://';
const FIXED_PORT = '8080';
const FIXED_UPLOAD_PATH = '/smallcell/FileUploadService';
const FIXED_DOWNLOAD_PATH = '/smallcell/FileDownloadService';
const DEFAULT_MAX_FILE_SIZE = 1073741824; // 1 GiB
const DEFAULT_HOST = '127.0.0.1'; // 默认服务器 IP（可编辑）

const cardTitleStyle: React.CSSProperties = {
	fontSize: 14,
	fontWeight: 600,
};

// 从已存的 BaseURL（如 http://127.0.0.1:8080）反解出纯 IP / 主机名，
// 用于把后端存量值还原到“只显示 IP”的输入框里。容错处理协议、端口、路径。
function extractHost(baseURL: string | undefined): string {
	if (!baseURL) return '';
	let s = baseURL.trim();
	s = s.replace(/^https?:\/\//i, ''); // 去协议
	s = s.split('/')[0]; // 去路径
	s = s.split(':')[0]; // 去端口
	return s.trim();
}

// 把用户填的 IP / 主机名拼成固定协议+端口的 BaseURL；空输入返回空串。
function buildBaseURL(host: string): string {
	const h = (host || '').trim();
	if (!h) return '';
	return `${FIXED_PROTOCOL}${h}:${FIXED_PORT}`;
}

// 校验 IP（IPv4）或主机名，禁止把协议/端口/路径塞进来。
function isValidHost(host: string): boolean {
	if (!host || /[\s/:]/.test(host)) return false;
	const ipv4 = host.match(/^(\d{1,3})\.(\d{1,3})\.(\d{1,3})\.(\d{1,3})$/);
	if (ipv4) {
		return ipv4.slice(1).every((o) => Number(o) >= 0 && Number(o) <= 255);
	}
	return /^[a-zA-Z0-9]([a-zA-Z0-9-]*[a-zA-Z0-9])?(\.[a-zA-Z0-9]([a-zA-Z0-9-]*[a-zA-Z0-9])?)*$/.test(host);
}

export default function TransferSettings({ form }: TransferSettingsProps) {
	const t = useT();

	// 监听上传 BaseURL：它是“服务器 IP”输入框的真值源（存全 URL，显示只给 IP）。
	const uploadBaseURL = Form.useWatch('uploadBaseURL', form) as string | undefined;
	const serverHost = extractHost(uploadBaseURL);

	// 收敛副作用：把所有“写死项”强制对齐，且对后端存量里非 8080 / 带 https 的旧值做归一化。
	// 幂等回写（仅在与当前值不同时 setFieldsValue），既能纠正存量，又不会和父组件的
	// 异步加载（resetFields + setFieldsValue）互相打架——每次加载后本效应再收敛一次。
	useEffect(() => {
		const normalized = uploadBaseURL ? buildBaseURL(extractHost(uploadBaseURL)) : '';
		const patch: Record<string, unknown> = {};
		if (uploadBaseURL !== undefined && uploadBaseURL !== normalized) patch.uploadBaseURL = normalized;
		// 下发地址与上传共用同一个 IP
		if (form.getFieldValue('downloadBaseURL') !== normalized) patch.downloadBaseURL = normalized;
		// 上传 / 下发路径写死
		if (form.getFieldValue('uploadPath') !== FIXED_UPLOAD_PATH) patch.uploadPath = FIXED_UPLOAD_PATH;
		if (form.getFieldValue('downloadPath') !== FIXED_DOWNLOAD_PATH) patch.downloadPath = FIXED_DOWNLOAD_PATH;
		if (Object.keys(patch).length > 0) form.setFieldsValue(patch);
	}, [uploadBaseURL, form]);

	const uploadURL = serverHost ? `${buildBaseURL(serverHost)}${FIXED_UPLOAD_PATH}` : '—';
	const downloadURL = serverHost ? `${buildBaseURL(serverHost)}${FIXED_DOWNLOAD_PATH}` : '—';

	return (
		<Form
			form={form}
			layout="vertical"
			size="small"
			initialValues={{
				uploadBaseURL: buildBaseURL(DEFAULT_HOST),
				downloadBaseURL: buildBaseURL(DEFAULT_HOST),
				uploadPath: FIXED_UPLOAD_PATH,
				downloadPath: FIXED_DOWNLOAD_PATH,
				uploadMaxFileSize: DEFAULT_MAX_FILE_SIZE,
			}}
		>
			{/* 文件传输服务器：唯一可填项 —— 服务器 IP */}
			<Card
				size="small"
				title={<span style={cardTitleStyle}>{t('system.transfer.serverSection')}</span>}
				style={{ marginBottom: 16 }}
			>
				<Form.Item
					name="uploadBaseURL"
					label={t('system.transfer.serverIP')}
					extra={t('system.transfer.serverIPHelp')}
					// 存全 URL（http://{ip}:8080），输入框只显示 / 只编辑 IP
					getValueProps={(value) => ({ value: extractHost(value as string | undefined) })}
					normalize={(input) => buildBaseURL(extractHost(input as string))}
					rules={[
						{ required: true, message: t('system.transfer.serverIPRequired') },
						{
							validator: (_, value) => {
								const host = extractHost(value as string | undefined);
								if (!host) return Promise.resolve(); // 空值交给 required
								return isValidHost(host)
									? Promise.resolve()
									: Promise.reject(new Error(t('system.transfer.serverIPInvalid')));
							},
						},
					]}
				>
					{/* antd6 addonBefore/addonAfter 已废弃：受控 AddonInput（Space.Compact +
					    InputAddon）复刻 http:// 前缀 + :8080 后缀盒子，保留 Form.Item 受控绑定。 */}
					<AddonInput addonBefore={FIXED_PROTOCOL} addonAfter={`:${FIXED_PORT}`} placeholder={DEFAULT_HOST} />
				</Form.Item>

				{/* 设备最终拿到的完整地址预览（只读） */}
				<Typography.Paragraph type="secondary" style={{ marginBottom: 0 }}>
					<div>
						{t('system.transfer.uploadURLPreview')}：<Typography.Text code copyable={serverHost ? { text: uploadURL } : false}>{uploadURL}</Typography.Text>
					</div>
					<div>
						{t('system.transfer.downloadURLPreview')}：<Typography.Text code copyable={serverHost ? { text: downloadURL } : false}>{downloadURL}</Typography.Text>
					</div>
				</Typography.Paragraph>
			</Card>

			{/* 上传服务：路径写死 + 最大文件大小可编辑 */}
			<Card
				size="small"
				title={<span style={cardTitleStyle}>{t('system.transfer.uploadSection')}</span>}
				style={{ marginBottom: 16 }}
			>
				<Space orientation="vertical" style={{ width: '100%' }} size={12}>
					<Form.Item
						name="uploadPath"
						label={t('system.transfer.uploadPath')}
						extra={t('system.transfer.pathFixedHelp')}
					>
						<Input disabled />
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
					name="downloadPath"
					label={t('system.transfer.downloadPath')}
					extra={t('system.transfer.pathFixedHelp')}
				>
					<Input disabled />
				</Form.Item>
				{/* 下发 BaseURL 与上传共用同一 IP，由收敛副作用自动写入，无需用户填写 */}
				<Form.Item name="downloadBaseURL" hidden>
					<Input />
				</Form.Item>
			</Card>

			<Typography.Paragraph type="secondary" style={{ marginTop: 12, marginBottom: 0 }}>
				{t('system.transfer.runtimeHint')}
			</Typography.Paragraph>
		</Form>
	);
}
