import { useCallback, useEffect, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import {
  Button,
  Card,
  ConfigProvider,
  Form,
  Input,
  InputNumber,
  Layout,
  Modal,
  Space,
  Switch,
  Table,
  Tag,
  Typography,
  message,
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { PlusOutlined, ReloadOutlined } from '@ant-design/icons';

import AppSidebar from '@/layouts/AppSidebar';
import { useTheme } from '@/hooks/useTheme';
import { useInboundOptions } from '@/api/queries/useInboundOptions';
import { HttpUtil } from '@/utils';

interface Call {
  id: number;
  url: string;
  enabled: boolean;
  maxConfigs: number;
  assigned: number;
  overLimit: number;
}

interface Proxy {
  inboundId: number;
  listenPort: number;
  enabled: boolean;
  state: string;
  error: string;
}

interface Assignment {
  inboundId: number;
  email: string;
  callId: number | null;
  enabled: boolean;
}

interface CallForm {
  url: string;
  enabled: boolean;
  maxConfigs?: number | null;
}

const JSON_HEADERS = { headers: { 'Content-Type': 'application/json' } } as const;

export default function VkTurnPage() {
  const { t } = useTranslation();
  const { antdThemeConfig } = useTheme();
  const [messageApi, messageContext] = message.useMessage();
  const [modalApi, modalContext] = Modal.useModal();
  const { data: inbounds = [] } = useInboundOptions();
  const [calls, setCalls] = useState<Call[]>([]);
  const [proxies, setProxies] = useState<Proxy[]>([]);
  const [assignments, setAssignments] = useState<Assignment[]>([]);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [editingCall, setEditingCall] = useState<Call | null>(null);
  const [callOpen, setCallOpen] = useState(false);
  const [callForm] = Form.useForm<CallForm>();
  const [proxyDrafts, setProxyDrafts] = useState<
    Record<number, { listenPort: number; enabled: boolean }>
  >({});

  const refresh = useCallback(async () => {
    setLoading(true);
    try {
      const [callMsg, proxyMsg, assignmentMsg] = await Promise.all([
        HttpUtil.get<Call[]>('/panel/api/vkturn/calls', undefined, { silent: true }),
        HttpUtil.get<Proxy[]>('/panel/api/vkturn/proxies', undefined, { silent: true }),
        HttpUtil.get<Assignment[]>('/panel/api/vkturn/assignments', undefined, { silent: true }),
      ]);
      const failed = [callMsg, proxyMsg, assignmentMsg].find((item) => !item.success);
      if (failed) throw new Error(failed.msg || t('pages.vkTurn.loadFailed'));
      setCalls(Array.isArray(callMsg.obj) ? callMsg.obj : []);
      const nextProxies = Array.isArray(proxyMsg.obj) ? proxyMsg.obj : [];
      setProxies(nextProxies);
      setProxyDrafts(
        Object.fromEntries(
          nextProxies.map((p) => [p.inboundId, { listenPort: p.listenPort, enabled: p.enabled }]),
        ),
      );
      setAssignments(Array.isArray(assignmentMsg.obj) ? assignmentMsg.obj : []);
    } catch (error) {
      messageApi.error(error instanceof Error ? error.message : t('pages.vkTurn.loadFailed'));
    } finally {
      setLoading(false);
    }
  }, [messageApi, t]);

  useEffect(() => {
    const timer = window.setTimeout(() => void refresh(), 0);
    return () => window.clearTimeout(timer);
  }, [refresh]);

  const wireguardInbounds = useMemo(
    () =>
      inbounds.filter(
        (ib) => ib.protocol === 'wireguard' && ib.nodeId == null && (ib.port || 0) > 0,
      ),
    [inbounds],
  );
  const inboundNames = useMemo(
    () => new Map(inbounds.map((ib) => [ib.id, ib.remark || ib.tag || String(ib.id)])),
    [inbounds],
  );
  const callUrls = useMemo(() => new Map(calls.map((call) => [call.id, call.url])), [calls]);

  function openCall(call: Call | null) {
    setEditingCall(call);
    callForm.resetFields();
    callForm.setFieldsValue({
      url: call?.url || '',
      enabled: call?.enabled ?? true,
      maxConfigs: call?.maxConfigs || undefined,
    });
    setCallOpen(true);
  }

  async function saveCall() {
    const values = await callForm.validateFields();
    setSaving(true);
    try {
      const body = {
        url: values.url.trim(),
        enabled: values.enabled,
        maxConfigs: values.maxConfigs ?? 0,
      };
      const msg = editingCall
        ? await HttpUtil.put(`/panel/api/vkturn/calls/${editingCall.id}`, body, JSON_HEADERS)
        : await HttpUtil.post('/panel/api/vkturn/calls', body, JSON_HEADERS);
      if (!msg.success) return;
      setCallOpen(false);
      await refresh();
    } finally {
      setSaving(false);
    }
  }

  async function deleteCall(call: Call) {
    const msg = await HttpUtil.delete(`/panel/api/vkturn/calls/${call.id}`);
    if (msg.success) await refresh();
  }

  async function saveProxy(inboundId: number) {
    const draft = proxyDrafts[inboundId];
    if (!draft) return;
    setSaving(true);
    try {
      const msg = await HttpUtil.put(`/panel/api/vkturn/proxies/${inboundId}`, draft, JSON_HEADERS);
      if (msg.success) await refresh();
    } finally {
      setSaving(false);
    }
  }

  const callColumns: ColumnsType<Call> = [
    {
      title: t('pages.vkTurn.callUrl'),
      dataIndex: 'url',
      render: (url: string) => (
        <Typography.Text copyable ellipsis={{ tooltip: url }} style={{ maxWidth: 440 }}>
          {url}
        </Typography.Text>
      ),
    },
    {
      title: t('pages.vkTurn.enabled'),
      dataIndex: 'enabled',
      width: 95,
      render: (enabled: boolean) => (
        <Tag color={enabled ? 'green' : 'default'}>{enabled ? t('yes') : t('no')}</Tag>
      ),
    },
    {
      title: t('pages.vkTurn.configs'),
      width: 170,
      render: (_, call) => (
        <span>
          {call.assigned} / {call.maxConfigs === 0 ? '∞' : call.maxConfigs}
          {call.overLimit > 0 && (
            <Tag color="red" style={{ marginInlineStart: 8 }}>
              +{call.overLimit}
            </Tag>
          )}
        </span>
      ),
    },
    {
      title: t('pages.vkTurn.actions'),
      width: 170,
      render: (_, call) => (
        <Space>
          <Button size="small" onClick={() => openCall(call)}>
            {t('edit')}
          </Button>
          <Button
            size="small"
            danger
            onClick={() =>
              modalApi.confirm({
                title: t('pages.vkTurn.deleteCall'),
                content: call.url,
                okText: t('delete'),
                okType: 'danger',
                onOk: () => deleteCall(call),
              })
            }
          >
            {t('delete')}
          </Button>
        </Space>
      ),
    },
  ];

  const proxyColumns: ColumnsType<{ inboundId: number; name: string; proxy?: Proxy }> = [
    { title: t('pages.vkTurn.inbound'), dataIndex: 'name' },
    {
      title: t('pages.vkTurn.udpPort'),
      width: 145,
      render: (_, row) => (
        <InputNumber
          min={1}
          max={65535}
          value={proxyDrafts[row.inboundId]?.listenPort ?? row.proxy?.listenPort}
          onChange={(value) =>
            setProxyDrafts((prev) => ({
              ...prev,
              [row.inboundId]: {
                listenPort: value || 0,
                enabled: prev[row.inboundId]?.enabled ?? row.proxy?.enabled ?? false,
              },
            }))
          }
        />
      ),
    },
    {
      title: t('pages.vkTurn.enabled'),
      width: 100,
      render: (_, row) => (
        <Switch
          checked={proxyDrafts[row.inboundId]?.enabled ?? row.proxy?.enabled ?? false}
          onChange={(enabled) =>
            setProxyDrafts((prev) => ({
              ...prev,
              [row.inboundId]: {
                listenPort: prev[row.inboundId]?.listenPort ?? row.proxy?.listenPort ?? 0,
                enabled,
              },
            }))
          }
        />
      ),
    },
    {
      title: t('pages.vkTurn.status'),
      render: (_, row) => (
        <Space direction="vertical" size={0}>
          <Tag color={row.proxy?.state === 'running' ? 'green' : 'default'}>
            {row.proxy?.state || '—'}
          </Tag>
          {row.proxy?.error && <Typography.Text type="danger">{row.proxy.error}</Typography.Text>}
        </Space>
      ),
    },
    {
      title: '',
      width: 100,
      render: (_, row) => (
        <Button
          size="small"
          loading={saving}
          disabled={!proxyDrafts[row.inboundId]?.listenPort}
          onClick={() => saveProxy(row.inboundId)}
        >
          {t('save')}
        </Button>
      ),
    },
  ];

  const proxyRows = wireguardInbounds.map((ib) => ({
    inboundId: ib.id,
    name: ib.remark || ib.tag || String(ib.id),
    proxy: proxies.find((p) => p.inboundId === ib.id),
  }));
  const assignmentColumns: ColumnsType<Assignment> = [
    {
      title: t('pages.vkTurn.inbound'),
      render: (_, row) => inboundNames.get(row.inboundId) || String(row.inboundId),
    },
    { title: t('pages.vkTurn.client'), dataIndex: 'email' },
    {
      title: t('pages.vkTurn.callUrl'),
      render: (_, row) =>
        row.callId == null ? (
          <Tag>{t('pages.vkTurn.waiting')}</Tag>
        ) : (
          <Typography.Text
            ellipsis={{ tooltip: callUrls.get(row.callId) }}
            style={{ maxWidth: 420 }}
          >
            {callUrls.get(row.callId) || `#${row.callId}`}
          </Typography.Text>
        ),
    },
  ];

  return (
    <ConfigProvider theme={antdThemeConfig}>
      {messageContext}
      {modalContext}
      <Layout>
        <AppSidebar />
        <Layout className="content-shell">
          <Layout.Content id="content-layout" className="content-area">
            <Space direction="vertical" size="middle" style={{ width: '100%' }}>
              <Card
                size="small"
                title={t('menu.vkTurn')}
                extra={
                  <Space>
                    <Button
                      icon={<ReloadOutlined />}
                      onClick={() => void refresh()}
                      loading={loading}
                    >
                      {t('refresh')}
                    </Button>
                    <Button type="primary" icon={<PlusOutlined />} onClick={() => openCall(null)}>
                      {t('pages.vkTurn.addCall')}
                    </Button>
                  </Space>
                }
              >
                <Table
                  rowKey="id"
                  size="small"
                  loading={loading}
                  dataSource={calls}
                  columns={callColumns}
                  pagination={false}
                  scroll={{ x: 750 }}
                />
              </Card>
              <Card size="small" title={t('pages.vkTurn.proxies')}>
                <Table
                  rowKey="inboundId"
                  size="small"
                  loading={loading}
                  dataSource={proxyRows}
                  columns={proxyColumns}
                  pagination={false}
                  scroll={{ x: 700 }}
                />
              </Card>
              <Card size="small" title={t('pages.vkTurn.assignments')}>
                <Table
                  rowKey={(row) => `${row.inboundId}:${row.email}`}
                  size="small"
                  loading={loading}
                  dataSource={assignments.filter((row) => row.enabled)}
                  columns={assignmentColumns}
                  pagination={{ pageSize: 20 }}
                  scroll={{ x: 700 }}
                />
              </Card>
            </Space>
          </Layout.Content>
        </Layout>
      </Layout>
      <Modal
        open={callOpen}
        title={editingCall ? t('pages.vkTurn.editCall') : t('pages.vkTurn.addCall')}
        okText={t('save')}
        confirmLoading={saving}
        onOk={() => void saveCall()}
        onCancel={() => setCallOpen(false)}
        destroyOnHidden
      >
        <Form form={callForm} layout="vertical" initialValues={{ enabled: true }}>
          <Form.Item
            name="url"
            label={t('pages.vkTurn.callUrl')}
            rules={[
              { required: true },
              {
                pattern: /^https:\/\/vk\.(?:com|ru)\/call\/join\/[^\s/?#]+\/?(?:\?[^\s#]*)?$/,
                message: t('pages.vkTurn.invalidUrl'),
              },
            ]}
          >
            <Input />
          </Form.Item>
          <Form.Item
            name="maxConfigs"
            label={t('pages.vkTurn.limit')}
            extra={t('pages.vkTurn.unlimitedHint')}
          >
            <InputNumber min={1} precision={0} style={{ width: '100%' }} />
          </Form.Item>
          <Form.Item name="enabled" label={t('pages.vkTurn.enabled')} valuePropName="checked">
            <Switch />
          </Form.Item>
        </Form>
      </Modal>
    </ConfigProvider>
  );
}
