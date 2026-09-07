import React, { useState, useEffect, useCallback } from 'react';
import { Card, Table, Button, Modal, Form, Select, Tag, Space, Typography, Popconfirm, message } from 'antd';
import { LinkOutlined, ThunderboltOutlined, PlusOutlined, DeleteOutlined, CloudUploadOutlined } from '@ant-design/icons';
import { api } from '../../api';

const { Title, Text } = Typography;

export function PairingsPage() {
  const [pairings, setPairings] = useState<any[]>([]);
  const [accounts, setAccounts] = useState<any[]>([]);
  const [serverAccounts, setServerAccounts] = useState<any[]>([]);
  const [showAdd, setShowAdd] = useState(false);
  const [loading, setLoading] = useState(false);
  const [panelRole, setPanelRole] = useState<string>('');
  const [form] = Form.useForm();

  const load = useCallback(async () => {
    try {
      const [p, a, s] = await Promise.all([
        api.listPairings(),
        api.listAccounts(),
        api.listAccounts('SERVER'),
      ]);
      setPairings(p);
      setAccounts(a);
      setServerAccounts(s);
    } catch (e: any) {
      message.error('Failed to load pairings');
    }
  }, []);

  // Detect panel role and load initial data
  useEffect(() => {
    api.syncStatus().then(s => {
      setPanelRole(s.role === 'client' ? 'CLIENT' : 'SERVER');
    }).catch(() => {});
    load();
  }, [load]);

  const handleAdd = async (values: any) => {
    setLoading(true);
    try {
      await api.createPairing(Number(values.clientId), Number(values.serverId));
      message.success('Pairing created successfully');
      setShowAdd(false);
      form.resetFields();
      load();
    } catch (e: any) {
      message.error(e.message || 'Failed to create pairing');
    } finally {
      setLoading(false);
    }
  };

  const autoPair = async () => {
    try {
      const r = await api.autoPair();
      message.success(`Auto-paired ${r.paired} accounts`);
      load();
    } catch (e: any) {
      message.error(e.message || 'Auto-pair failed');
    }
  };

  const remove = async (id: number) => {
    try {
      await api.deletePairing(id);
      message.success('Pairing deleted');
      load();
    } catch (e: any) {
      message.error('Failed to delete pairing');
    }
  };

  const pushToServer = async (id: number) => {
    try {
      await api.remotePushSinglePairing(id);
      message.success('Pairing pushed to remote server');
    } catch (e: any) {
      message.error(e.message || 'Failed to push pairing to server');
    }
  };

  const openAddModal = () => {
    api.listAccounts('SERVER').then(setServerAccounts).catch(() => {});
    setShowAdd(true);
  };

  const timeAgo = (ts: string) => {
    if (!ts) return '—';
    const d = Date.now() - new Date(ts).getTime();
    if (d < 60000) return 'just now';
    if (d < 3600000) return Math.floor(d / 60000) + 'm ago';
    if (d < 86400000) return Math.floor(d / 3600000) + 'h ago';
    return Math.floor(d / 86400000) + 'd ago';
  };

  const columns = [
    {
      title: 'ID',
      dataIndex: 'id',
      key: 'id',
      render: (text: string) => <Text strong>{text}</Text>,
    },
    {
      title: 'Client',
      key: 'client',
      render: (_: any, r: any) => (
        <Space>
          <Tag color="purple">CLIENT</Tag>
          <Text type="secondary" className="font-mono">
            {r.client_account?.bale_user_id || r.client_account_id}
          </Text>
          {r.client_account?.display_name && (
            <Text type="secondary" style={{ fontSize: 11 }}>({r.client_account.display_name})</Text>
          )}
        </Space>
      ),
    },
    {
      title: 'Server',
      key: 'server',
      render: (_: any, r: any) => (
        <Space>
          <Tag color="blue">SERVER</Tag>
          <Text type="secondary" className="font-mono">
            {r.server_account?.bale_user_id || r.server_account_id}
          </Text>
          {r.server_account?.display_name && (
            <Text type="secondary" style={{ fontSize: 11 }}>({r.server_account.display_name})</Text>
          )}
        </Space>
      ),
    },
    {
      title: 'Status',
      key: 'status',
      render: (_: any, r: any) => (
        r.active ? <Tag color="success">Active</Tag> : <Tag color="default">Inactive</Tag>
      ),
    },
    {
      title: 'Created',
      dataIndex: 'created_at',
      key: 'created_at',
      render: (ts: string) => timeAgo(ts),
    },
    {
      title: 'Action',
      key: 'action',
      render: (_: any, r: any) => (
        <Space>
          <Button size="small" icon={<CloudUploadOutlined />} onClick={() => pushToServer(r.id)}
            style={{ color: '#52c41a', borderColor: '#52c41a' }}>
            Push
          </Button>
          <Popconfirm
            title="Delete the pairing"
            description="Are you sure to delete this pairing?"
            onConfirm={() => remove(r.id)}
            okText="Yes"
            cancelText="No"
          >
            <Button size="small" danger icon={<DeleteOutlined />}>Delete</Button>
          </Popconfirm>
        </Space>
      ),
    },
  ];

  const clients = accounts.filter((a: any) => a.role === 'CLIENT');

  return (
    <div className="space-y-6">
      <div className="flex justify-between items-end mb-8">
        <div>
          <Title level={2} style={{ margin: 0 }}>Pairings</Title>
          <Text type="secondary">
            Client ↔ Server account mappings
            {panelRole && <Tag color={panelRole === 'CLIENT' ? 'purple' : 'blue'} style={{ marginLeft: 8 }}>{panelRole} Panel</Tag>}
          </Text>
        </div>
        <Space>
          <Button icon={<ThunderboltOutlined />} onClick={autoPair}>
            Auto-Pair
          </Button>
          <Button type="primary" icon={<PlusOutlined />} onClick={openAddModal}>
            Create Pairing
          </Button>
        </Space>
      </div>

      <Card bordered={false} className="shadow-sm">
        <Table
          dataSource={pairings}
          columns={columns}
          rowKey="id"
          pagination={{ pageSize: 10 }}
          locale={{
            emptyText: (
              <div className="py-8">
                <LinkOutlined className="text-4xl text-slate-300 mb-2" />
                <p>No pairings yet</p>
              </div>
            )
          }}
        />
      </Card>

      <Modal
        title="Create Pairing"
        open={showAdd}
        onCancel={() => {
          setShowAdd(false);
          form.resetFields();
        }}
        footer={null}
      >
        <Form
          form={form}
          layout="vertical"
          onFinish={handleAdd}
          className="mt-4"
        >
          <Form.Item name="clientId" label="Client Account" rules={[{ required: true, message: 'Please select a client account' }]}>
            <Select placeholder="Select client...">
              {clients.map((c: any) => (
                <Select.Option key={c.id} value={c.id}>
                  #{c.id} — {c.bale_user_id} {c.display_name ? `(${c.display_name})` : ''}
                </Select.Option>
              ))}
            </Select>
          </Form.Item>
          <Form.Item name="serverId" label="Server Account" rules={[{ required: true, message: 'Please select a server account' }]}>
            <Select placeholder="Select server...">
              {serverAccounts.map((s: any) => (
                <Select.Option key={s.id} value={s.id}>
                  #{s.id} — {s.bale_user_id} {s.display_name ? `(${s.display_name})` : ''}
                </Select.Option>
              ))}
            </Select>
          </Form.Item>
          <Form.Item className="mb-0 flex justify-end">
            <Space>
              <Button onClick={() => setShowAdd(false)}>Cancel</Button>
              <Button type="primary" htmlType="submit" loading={loading}>
                Create
              </Button>
            </Space>
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
}
