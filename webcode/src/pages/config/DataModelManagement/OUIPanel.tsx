import { useCallback, useMemo, useState } from 'react';
import { Button, Form, Input, Modal, Space, Table, message } from 'antd';
import { PlusOutlined, ReloadOutlined } from '@ant-design/icons';
import type { ColumnsType } from 'antd/es/table';
import { useOUIList, useCreateOUI } from '@/hooks/api/useDataModels';
import type { OUIEntry, CreateOUIRequest } from '@/services/api/datamodelApi';

export default function OUIPanel() {
  const [modalOpen, setModalOpen] = useState(false);
  const [searchText, setSearchText] = useState('');
  const [form] = Form.useForm();

  const { data: ouiList, isLoading, refetch } = useOUIList();
  const createMutation = useCreateOUI();

  const filteredList = useMemo(() => {
    if (!ouiList) return [];
    if (!searchText) return ouiList;
    const kw = searchText.toLowerCase();
    return ouiList.filter(
      (e) =>
        e.oui.toLowerCase().includes(kw) ||
        e.manufacturer.toLowerCase().includes(kw) ||
        e.shortName.toLowerCase().includes(kw)
    );
  }, [ouiList, searchText]);

  const handleCreate = useCallback(async () => {
    try {
      const values = await form.validateFields();
      const req: CreateOUIRequest = {
        oui: values.oui,
        manufacturer: values.manufacturer,
        shortName: values.shortName,
        country: values.country,
      };
      await createMutation.mutateAsync(req);
      void message.success('OUI entry created');
      setModalOpen(false);
      form.resetFields();
    } catch {
      // validation failed
    }
  }, [form, createMutation]);

  const columns = useMemo(
    (): ColumnsType<OUIEntry> => [
      {
        title: 'OUI',
        dataIndex: 'oui',
        key: 'oui',
        width: 120,
        render: (val: string) => <span style={{ fontFamily: 'monospace' }}>{val}</span>,
      },
      {
        title: 'Manufacturer',
        dataIndex: 'manufacturer',
        key: 'manufacturer',
        width: 200,
      },
      {
        title: 'Short Name',
        dataIndex: 'shortName',
        key: 'shortName',
        width: 120,
      },
      {
        title: 'Country',
        dataIndex: 'country',
        key: 'country',
        width: 100,
        render: (val: string) => val || '-',
      },
      {
        title: 'Created',
        dataIndex: 'createdAt',
        key: 'createdAt',
        width: 160,
        render: (val: string) => (val ? new Date(val).toLocaleString('zh-CN') : '-'),
      },
    ],
    []
  );

  return (
    <>
      <Space style={{ marginBottom: 16 }}>
        <Input.Search
          placeholder="Search OUI / Manufacturer"
          allowClear
          style={{ width: 300 }}
          onSearch={(val) => setSearchText(val)}
          onChange={(e) => {
            if (!e.target.value) setSearchText('');
          }}
        />
        <Button icon={<ReloadOutlined />} onClick={() => void refetch()}>
          Refresh
        </Button>
        <Button type="primary" icon={<PlusOutlined />} onClick={() => setModalOpen(true)}>
          Add OUI
        </Button>
      </Space>

      <Table<OUIEntry>
        columns={columns}
        dataSource={filteredList}
        loading={isLoading}
        rowKey="oui"
        size="small"
        pagination={{
          pageSize: 20,
          showSizeChanger: true,
          showTotal: (t) => `Total ${t} entries`,
        }}
      />

      <Modal
        title="Add OUI Entry"
        open={modalOpen}
        onCancel={() => setModalOpen(false)}
        onOk={() => void handleCreate()}
        confirmLoading={createMutation.isPending}
        destroyOnClose
      >
        <Form form={form} layout="vertical" style={{ marginTop: 16 }}>
          <Form.Item
            name="oui"
            label="OUI (6-char hex)"
            rules={[
              { required: true, message: 'OUI is required' },
              { pattern: /^[0-9A-Fa-f]{6}$/, message: 'OUI must be 6 hex characters' },
            ]}
          >
            <Input placeholder="e.g. 00259E" style={{ fontFamily: 'monospace' }} maxLength={6} />
          </Form.Item>
          <Form.Item
            name="manufacturer"
            label="Manufacturer"
            rules={[{ required: true, message: 'Manufacturer is required' }]}
          >
            <Input placeholder="e.g. Huawei Technologies Co., Ltd." />
          </Form.Item>
          <Form.Item
            name="shortName"
            label="Short Name"
            rules={[{ required: true, message: 'Short name is required' }]}
          >
            <Input placeholder="e.g. Huawei" />
          </Form.Item>
          <Form.Item name="country" label="Country">
            <Input placeholder="e.g. CN" />
          </Form.Item>
        </Form>
      </Modal>
    </>
  );
}
