import React, { useState, useEffect, useCallback, useRef, useMemo } from 'react';
import {
  Card, Typography, Select, Button, Space, Row, Col, Tag, message,
  Descriptions, Badge, Alert, Upload, Divider, Input, Table, Progress,
  Popover, Switch, Popconfirm
} from 'antd';
import {
  SyncOutlined, CloudSyncOutlined, CheckCircleOutlined, DownloadOutlined,
  UploadOutlined, DatabaseOutlined, ExclamationCircleOutlined, LinkOutlined,
  GlobalOutlined, SafetyCertificateOutlined, ApiOutlined,
  PlayCircleOutlined, StopOutlined, DashboardOutlined,
  RocketOutlined, SearchOutlined
} from '@ant-design/icons';
import { useTheme, THEMES, MODES } from '../../ThemeContext';
import { api } from '../../api';

const { Title, Text } = Typography;

const SWATCH_COLORS: Record<string, [string, string]> = {
  indigo:  ['#6366f1', '#818cf8'],
  violet:  ['#8b5cf6', '#a78bfa'],
  emerald: ['#10b981', '#34d399'],
  rose:    ['#f43f5e', '#fb7185'],
  amber:   ['#f59e0b', '#fbbf24'],
  cyan:    ['#06b6d4', '#22d3ee'],
};

export function SettingsPage() {
  const { settings, update } = useTheme();
  const [syncStatus, setSyncStatus] = useState<any>(null);
  const [syncing, setSyncing] = useState(false);
  const [pushing, setPushing] = useState<string | null>(null);
  const [backingUp, setBackingUp] = useState(false);
  const [restoring, setRestoring] = useState(false);
  const fileInputRef = useRef<HTMLInputElement>(null);
  const [panelRole, setPanelRole] = useState<string>('');
  const [serverURLs, setServerURLs] = useState<string[]>([]);
  const [remoteServerURL, setRemoteServerURL] = useState<string>('');
  const [baleConstants, setBaleConstants] = useState<any>(null);
  const [baleSyncing, setBaleSyncing] = useState(false);
  const [routing, setRouting] = useState({ dns_primary: '', dns_secondary: '', bypass_domains: '' });
  const [routingSaving, setRoutingSaving] = useState(false);

  // S3 Cloud Persistence state
  const [s3Status, setS3Status] = useState<any>(null);
  const [s3BackupLoading, setS3BackupLoading] = useState(false);
  const [s3RestoreLoading, setS3RestoreLoading] = useState(false);

  const loadS3Status = useCallback(async () => {
    try {
      const res = await api.getS3Status();
      setS3Status(res);
    } catch {
      // ignore if not supported
    }
  }, []);

  useEffect(() => {
    loadS3Status();
  }, [loadS3Status]);

  const handleS3BackupNow = async () => {
    setS3BackupLoading(true);
    try {
      await api.triggerS3Backup();
      message.success('Database successfully backed up to Cellar S3!');
      await loadS3Status();
    } catch (e: any) {
      message.error(e.message || 'S3 backup failed');
    } finally {
      setS3BackupLoading(false);
    }
  };

  const handleS3RestoreNow = async () => {
    setS3RestoreLoading(true);
    try {
      const res = await api.triggerS3Restore();
      if (res?.restored) {
        message.success('Database successfully restored from S3!');
      } else {
        message.info(res?.message || 'No backup found in S3');
      }
      await loadS3Status();
    } catch (e: any) {
      message.error(e.message || 'S3 restore failed');
    } finally {
      setS3RestoreLoading(false);
    }
  };

  const [resetting, setResetting] = useState(false);
  const [remoteResetting, setRemoteResetting] = useState(false);

  const handleResetDatabase = async () => {
    setResetting(true);
    try {
      const res = await api.dbReset();
      message.success(res?.message || 'Database reset successfully: accounts, pairings, and logs cleared.');
      loadSyncStatus();
      loadS3Status();
    } catch (e: any) {
      message.error(e.message || 'Database reset failed');
    } finally {
      setResetting(false);
    }
  };

  const handleResetRemoteDatabase = async () => {
    setRemoteResetting(true);
    try {
      const res = await api.remoteDBReset();
      message.success(res?.message || 'Remote server database reset successfully.');
      loadSyncStatus();
    } catch (e: any) {
      message.error(e.message || 'Remote database reset failed');
    } finally {
      setRemoteResetting(false);
    }
  };

  // DNS Speed Benchmark state
  const [benchmarkStatus, setBenchmarkStatus] = useState<any>(null);
  const [benchmarkLoading, setBenchmarkLoading] = useState(false);
  const [dnsSearch, setDnsSearch] = useState('');
  const [onlineOnly, setOnlineOnly] = useState(true);

  const pollBenchmark = useCallback(async () => {
    try {
      const res = await api.dnsBenchmarkStatus();
      setBenchmarkStatus(res);
      return res;
    } catch {
      return null;
    }
  }, []);

  useEffect(() => {
    pollBenchmark();
  }, [pollBenchmark]);

  // Polling loop when benchmark is active
  useEffect(() => {
    let timer: any = null;
    if (benchmarkStatus?.running) {
      timer = setInterval(async () => {
        const res = await pollBenchmark();
        if (res && !res.running) {
          clearInterval(timer);
        }
      }, 1000);
    }
    return () => {
      if (timer) clearInterval(timer);
    };
  }, [benchmarkStatus?.running, pollBenchmark]);

  const handleStartBenchmark = async () => {
    setBenchmarkLoading(true);
    try {
      await api.dnsBenchmarkStart();
      message.success('DNS benchmark started — testing servers against Google and Bale Meet gateways...');
      await pollBenchmark();
    } catch (e: any) {
      message.error(e.message || 'Failed to start benchmark');
    } finally {
      setBenchmarkLoading(false);
    }
  };

  const handleStopBenchmark = async () => {
    try {
      await api.dnsBenchmarkStop();
      message.info('DNS benchmark stopped');
      await pollBenchmark();
    } catch (e: any) {
      message.error(e.message || 'Failed to stop benchmark');
    }
  };

  const handleApplyDNS = async (primary: string, secondary: string) => {
    const newRouting = {
      dns_primary: primary,
      dns_secondary: secondary,
      bypass_domains: routing.bypass_domains,
    };
    try {
      await api.updateRoutingSettings(newRouting);
      setRouting(newRouting);
      message.success(`Applied DNS: Primary=${primary} | Secondary=${secondary} — active traffic routing updated`);
    } catch (e: any) {
      message.error(e.message || 'Failed to apply DNS');
    }
  };

  const loadSyncStatus = useCallback(async () => {
    try {
      const status = await api.syncStatus();
      setSyncStatus(status);
      setPanelRole(status.role === 'client' ? 'CLIENT' : 'SERVER');
    } catch { /* ignore */ }
    try {
      const stats = await api.getStats();
      if (stats.server_urls) setServerURLs(stats.server_urls);
      if (stats.remote_server_url) setRemoteServerURL(stats.remote_server_url);
    } catch { /* ignore */ }
  }, []);

  const loadBaleConstants = useCallback(async () => {
    try {
      const data = await api.getBaleConstants();
      setBaleConstants(data);
    } catch { /* ignore */ }
  }, []);

  const loadRoutingSettings = useCallback(async () => {
    try {
      const data = await api.getRoutingSettings();
      setRouting({
        dns_primary: data.dns_primary || '',
        dns_secondary: data.dns_secondary || '',
        bypass_domains: data.bypass_domains || '',
      });
    } catch { /* ignore */ }
  }, []);

  useEffect(() => { loadSyncStatus(); loadBaleConstants(); loadRoutingSettings(); }, [loadSyncStatus, loadBaleConstants, loadRoutingSettings]);

  const handleBaleSync = async () => {
    setBaleSyncing(true);
    try {
      const data = await api.syncBaleConstants();
      if (data.status === 'success') {
        setBaleConstants(data);
        message.success('Bale constants synchronized from upstream');
      } else {
        message.error(data.message || 'Sync failed');
      }
    } catch (e: any) {
      message.error(e.message || 'Failed to sync Bale constants');
    } finally {
      setBaleSyncing(false);
    }
  };

  const handleManualSync = async () => {
    setSyncing(true);
    try {
      // 1. Push all local accounts to the remote server
      const pushRes = await api.remoteSyncPushAccounts();
      if (pushRes.pushed > 0) {
        message.success(`Pushed ${pushRes.pushed} accounts to remote server`);
      }

      // 2. Sync audit events (local -> remote)
      try {
        const localEvents = await api.syncPull(0);
        if (localEvents.events && localEvents.events.length > 0) {
          await api.remoteSyncPush(localEvents.events);
        }
      } catch (e) { console.warn('Audit sync push failed', e); }

      message.success('Synchronized with remote server successfully');
      await loadSyncStatus();
    } catch (e: any) {
      message.error(e.message || 'Sync failed');
    } finally {
      setSyncing(false);
    }
  };


  const topOnlineServers = useMemo(() => {
    if (!benchmarkStatus?.results) return [];
    return benchmarkStatus.results.filter((r: any) => r.online);
  }, [benchmarkStatus?.results]);

  const filteredResults = useMemo(() => {
    if (!benchmarkStatus?.results) return [];
    let list = benchmarkStatus.results;
    if (onlineOnly) {
      list = list.filter((r: any) => r.online);
    }
    if (dnsSearch.trim()) {
      const q = dnsSearch.trim().toLowerCase();
      list = list.filter((r: any) => r.ip.toLowerCase().includes(q));
    }
    return list;
  }, [benchmarkStatus?.results, onlineOnly, dnsSearch]);

  const dnsColumns = [
    {
      title: 'Rank',
      key: 'rank',
      width: 75,
      render: (_: any, r: any) => {
        if (!r.online) return <Tag color="default">Off</Tag>;
        const colors: Record<number, string> = { 1: '#f59e0b', 2: '#94a3b8', 3: '#d97706' };
        return (
          <Tag color={colors[r.rank] || 'blue'} className="font-bold text-xs">
            #{r.rank}
          </Tag>
        );
      },
    },
    {
      title: 'DNS Server IP',
      dataIndex: 'ip',
      key: 'ip',
      render: (ip: string) => (
        <Space size="small">
          <Text strong className="font-mono text-sm">{ip}</Text>
          {routing.dns_primary === ip && <Tag color="green">Primary</Tag>}
          {routing.dns_secondary === ip && <Tag color="cyan">Secondary</Tag>}
        </Space>
      ),
    },
    {
      title: 'Global Avg',
      key: 'global_avg_ms',
      sorter: (a: any, b: any) => (a.global_avg_ms > 0 ? a.global_avg_ms : 9999) - (b.global_avg_ms > 0 ? b.global_avg_ms : 9999),
      render: (_: any, r: any) => {
        if (!r.online || r.global_avg_ms < 0) return <Text type="secondary">—</Text>;
        const color = r.global_avg_ms < 60 ? 'green' : r.global_avg_ms < 120 ? 'blue' : r.global_avg_ms < 200 ? 'orange' : 'red';
        return <Tag color={color} className="font-semibold">{r.global_avg_ms} ms</Tag>;
      },
    },
    {
      title: 'Bale Meet Gateways',
      key: 'bale_avg_ms',
      sorter: (a: any, b: any) => (a.bale_avg_ms > 0 ? a.bale_avg_ms : 9999) - (b.bale_avg_ms > 0 ? b.bale_avg_ms : 9999),
      render: (_: any, r: any) => {
        if (!r.online || r.bale_avg_ms < 0) return <Text type="secondary">—</Text>;
        const content = (
          <div className="text-xs p-1 space-y-1 font-mono">
            <div className="font-bold border-b pb-1 mb-1">Meet Gateways (meet-gwbm[1..6].ble.ir):</div>
            {[1, 2, 3, 4, 5, 6].map((i) => {
              const val = r.bale_gateways?.[`B${i}`];
              return (
                <div key={i} className="flex justify-between gap-4">
                  <span className="text-slate-500">Gateway {i}:</span>
                  <span className={val > 0 ? 'text-emerald-600 font-bold' : 'text-rose-500'}>
                    {val > 0 ? `${val} ms` : 'Failed'}
                  </span>
                </div>
              );
            })}
          </div>
        );
        return (
          <Popover content={content} title="Gateway Latency Breakdown" trigger="hover">
            <Tag color="cyan" className="cursor-pointer font-semibold">
              {r.bale_avg_ms} ms (B1-B6 ℹ)
            </Tag>
          </Popover>
        );
      },
    },
    {
      title: 'Google',
      key: 'google_avg_ms',
      render: (_: any, r: any) => {
        if (!r.online || r.google_avg_ms < 0) return <Text type="secondary">X</Text>;
        return <Text className="font-mono text-xs">{r.google_avg_ms} ms</Text>;
      },
    },
    {
      title: 'Reliability',
      dataIndex: 'reliability',
      key: 'reliability',
      sorter: (a: any, b: any) => a.reliability - b.reliability,
      render: (val: number, r: any) => {
        if (!r.online) return <Tag color="error">0%</Tag>;
        const color = val === 100 ? 'success' : val >= 80 ? 'processing' : 'warning';
        return <Tag color={color} className="font-semibold">{val}%</Tag>;
      },
    },
    {
      title: 'Score',
      dataIndex: 'score',
      key: 'score',
      sorter: (a: any, b: any) => a.score - b.score,
      render: (val: number, r: any) => {
        if (!r.online || val >= 1000000) return <Text type="secondary">—</Text>;
        return <Text className="font-mono text-xs">{val}</Text>;
      },
    },
    {
      title: 'Actions',
      key: 'actions',
      render: (_: any, r: any) => {
        if (!r.online) return null;
        return (
          <Space size="small">
            <Button
              size="small"
              type={routing.dns_primary === r.ip ? 'primary' : 'default'}
              onClick={() => handleApplyDNS(r.ip, routing.dns_secondary)}
              title="Set as Primary DNS"
            >
              Primary
            </Button>
            <Button
              size="small"
              type={routing.dns_secondary === r.ip ? 'primary' : 'default'}
              onClick={() => handleApplyDNS(routing.dns_primary, r.ip)}
              title="Set as Secondary DNS"
            >
              Secondary
            </Button>
          </Space>
        );
      },
    },
  ];

  return (
    <div className="space-y-6">
      <div className="mb-8">
        <Title level={2} style={{ margin: 0 }}>Settings</Title>
        <Text type="secondary">Customize your admin panel experience</Text>
      </div>

      <Card title="Appearance" bordered={false} className="shadow-sm mb-6">
        <div className="mb-8">
          <Text type="secondary" strong className="block mb-4 uppercase text-xs tracking-wider">Mode</Text>
          <div className="flex gap-4">
            {MODES.map((mode) => (
              <div
                key={mode}
                onClick={() => update('mode', mode as 'dark' | 'light')}
                className={`flex-1 p-4 rounded-xl border-2 transition-all cursor-pointer flex items-center justify-center gap-4 ${
                  settings.mode === mode
                    ? 'border-indigo-500 bg-indigo-500/10'
                    : 'border-transparent bg-slate-100 dark:bg-slate-800 hover:bg-slate-200 dark:hover:bg-slate-700'
                }`}
              >
                <span className="text-2xl">{mode === 'dark' ? '🌙' : '☀️'}</span>
                <div>
                  <div className={`text-sm font-semibold ${settings.mode === mode ? 'text-indigo-600 dark:text-indigo-400' : 'text-slate-700 dark:text-slate-300'}`}>
                    {mode === 'dark' ? 'Dark Mode' : 'Light Mode'}
                  </div>
                  <div className="text-xs text-slate-500">
                    {mode === 'dark' ? 'Easy on the eyes' : 'Bright and clean'}
                  </div>
                </div>
              </div>
            ))}
          </div>
        </div>
        <div>
          <Text type="secondary" strong className="block mb-4 uppercase text-xs tracking-wider">Accent Color</Text>
          <div className="grid grid-cols-3 sm:grid-cols-6 gap-4">
            {Object.entries(THEMES).map(([key, theme]) => {
              const [c1, c2] = SWATCH_COLORS[key];
              const isActive = settings.color === key;
              return (
                <div
                  key={key}
                  onClick={() => update('color', key)}
                  className={`flex flex-col items-center gap-2 p-3 rounded-xl cursor-pointer transition-all ${
                    isActive ? 'bg-slate-100 dark:bg-slate-800' : 'hover:bg-slate-50 dark:hover:bg-slate-800/50'
                  }`}
                >
                  <div
                    className={`w-10 h-10 rounded-full transition-transform ${isActive ? 'scale-110 shadow-lg ring-2 ring-offset-2 ring-offset-white dark:ring-offset-slate-900' : ''}`}
                    style={{ background: `linear-gradient(135deg, ${c1}, ${c2})` }}
                  />
                  <Text className={`text-xs ${isActive ? 'font-bold' : ''}`}>{theme.name}</Text>
                </div>
              );
            })}
          </div>
        </div>
      </Card>

      <Card title="Panel Options" bordered={false} className="shadow-sm mb-6">
        <Row gutter={[24, 24]}>
          <Col xs={24} md={12}>
            <Text type="secondary" strong className="block mb-2 uppercase text-xs tracking-wider">Auto-refresh Interval</Text>
            <Select className="w-full" size="large" value={settings.refreshInterval} onChange={(v) => update('refreshInterval', Number(v))}>
              <Select.Option value={3}>3 seconds</Select.Option>
              <Select.Option value={5}>5 seconds</Select.Option>
              <Select.Option value={10}>10 seconds</Select.Option>
              <Select.Option value={30}>30 seconds</Select.Option>
              <Select.Option value={60}>60 seconds</Select.Option>
            </Select>
          </Col>
          <Col xs={24} md={12}>
            <Text type="secondary" strong className="block mb-2 uppercase text-xs tracking-wider">Connection History</Text>
            <Select className="w-full" size="large" value={settings.historyLimit || 50} onChange={(v) => update('historyLimit', Number(v))}>
              <Select.Option value={25}>Last 25 sessions</Select.Option>
              <Select.Option value={50}>Last 50 sessions</Select.Option>
              <Select.Option value={100}>Last 100 sessions</Select.Option>
            </Select>
          </Col>
        </Row>
      </Card>

      {/* Bale Client Constants — Dynamic Upstream Parameter Extraction */}
      <Card
        title={<><CloudSyncOutlined className="mr-2" />Bale Client Constants</>}
        bordered={false}
        className="shadow-sm mb-6"
        extra={
          <Space>
            <Button
              type="primary"
              icon={<SyncOutlined spin={baleSyncing} />}
              loading={baleSyncing}
              onClick={handleBaleSync}
            >
              Sync from Bale
            </Button>
          </Space>
        }
      >
        <Alert
          message="Dynamic Upstream Parameter Extraction"
          description="Bale updates client protocol parameters with each release. Click 'Sync from Bale' to scrape the live web bundle (web.bale.ai) and hot-swap these constants — new connections immediately use the updated values without a restart."
          type="info"
          showIcon
          icon={<CloudSyncOutlined />}
          className="mb-4"
        />

        {baleConstants ? (
          <>
            <Descriptions column={{ xs: 1, md: 2 }} size="small" bordered>
              <Descriptions.Item label="App Version">
                <Tag color="blue" className="font-mono">{baleConstants.app_version || '—'}</Tag>
              </Descriptions.Item>
              <Descriptions.Item label="LiveKit SDK Version">
                <Tag color="geekblue" className="font-mono">{baleConstants.livekit_sdk_version || '—'}</Tag>
              </Descriptions.Item>
              <Descriptions.Item label="LiveKit Protocol">
                <Tag color="geekblue" className="font-mono">v{baleConstants.livekit_protocol_version || '—'}</Tag>
                <Text type="secondary" className="ml-2 text-xs">subprotocol: lk-protocol-{baleConstants.livekit_protocol_version || '?'}</Text>
              </Descriptions.Item>
              <Descriptions.Item label="Browser Version">
                <Tag color="cyan" className="font-mono">{baleConstants.browser_version || '—'}</Tag>
              </Descriptions.Item>
              <Descriptions.Item label="Bale WS URL" span={2}>
                <Text className="font-mono text-xs">{baleConstants.bale_ws_url || '—'}</Text>
              </Descriptions.Item>
              <Descriptions.Item label="Bale gRPC Base" span={2}>
                <Text className="font-mono text-xs">{baleConstants.bale_grpc_base || '—'}</Text>
              </Descriptions.Item>
              <Descriptions.Item label="LiveKit Origin">
                <Text className="font-mono text-xs">{baleConstants.livekit_origin || '—'}</Text>
              </Descriptions.Item>
              <Descriptions.Item label="Bale Web Origin">
                <Text className="font-mono text-xs">{baleConstants.bale_web_origin || '—'}</Text>
              </Descriptions.Item>
              <Descriptions.Item label="Web API Key" span={2}>
                <Text className="font-mono text-xs" style={{ wordBreak: 'break-all' }}>
                  {baleConstants.web_api_key ? `${baleConstants.web_api_key.slice(0, 12)}...${baleConstants.web_api_key.slice(-8)}` : '—'}
                </Text>
              </Descriptions.Item>
            </Descriptions>

            <div className="mt-3 text-xs text-slate-400">
              <Badge
                status={baleSyncing ? 'processing' : (baleConstants.last_synced_at ? 'success' : 'default')}
                text={
                  baleSyncing ? 'Extracting parameters from upstream...'
                    : baleConstants.last_synced_at
                      ? `Last synced: ${baleConstants.last_synced_at}`
                      : 'Never synced — using default values'
                }
              />
            </div>
          </>
        ) : (
          <Text type="secondary">Loading Bale client constants...</Text>
        )}
      </Card>

      {/* Application-Level DNS & Split-Tunneling */}
      <Card
        title={<><SafetyCertificateOutlined className="mr-2" />Application DNS &amp; Split-Tunneling</>}
        bordered={false}
        className="shadow-sm mb-6"
        extra={
          <Button
            type="primary"
            icon={<SyncOutlined spin={routingSaving} />}
            loading={routingSaving}
            onClick={async () => {
              setRoutingSaving(true);
              try {
                await api.updateRoutingSettings({
                  dns_primary: routing.dns_primary.trim(),
                  dns_secondary: routing.dns_secondary.trim(),
                  bypass_domains: routing.bypass_domains.trim(),
                });
                message.success('Routing settings applied — new connections use updated DNS & bypass rules instantly');
                await loadRoutingSettings();
              } catch (e: any) {
                message.error(e.message || 'Failed to save routing settings');
              } finally {
                setRoutingSaving(false);
              }
            }}
          >
            Apply
          </Button>
        }
      >
        <Alert
          message="Application-Level DNS Resolution & Request Splitting"
          description={
            <div>
              <p className="mb-1">All domain resolution and proxy traffic routing is performed through the DNS servers below, decoupled from the host OS resolver.</p>
              <p className="mb-0">Iranian-domestic domains (servers hosted on Iranian IPs) and the custom bypass list below route <strong>directly over the local network</strong>, bypassing the WebRTC tunnel. Bale's own servers are never bypassed and always use the tunnel + custom DNS.</p>
            </div>
          }
          type="info"
          showIcon
          icon={<ApiOutlined />}
          className="mb-4"
        />

        <Row gutter={[24, 16]}>
          <Col xs={24} md={12}>
            <Text type="secondary" strong className="block mb-2 uppercase text-xs tracking-wider">Primary DNS Server</Text>
            <Input
              size="large"
              placeholder="1.1.1.1"
              value={routing.dns_primary}
              onChange={(e) => setRouting({ ...routing, dns_primary: e.target.value })}
            />
          </Col>
          <Col xs={24} md={12}>
            <Text type="secondary" strong className="block mb-2 uppercase text-xs tracking-wider">Secondary DNS Server</Text>
            <Input
              size="large"
              placeholder="1.0.0.1"
              value={routing.dns_secondary}
              onChange={(e) => setRouting({ ...routing, dns_secondary: e.target.value })}
            />
          </Col>
          <Col xs={24}>
            <Text type="secondary" strong className="block mb-2 uppercase text-xs tracking-wider">
              Bypass Domains (comma-separated)
            </Text>
            <Input.TextArea
              rows={3}
              placeholder="example.com, bank.ir, my-site.ir"
              value={routing.bypass_domains}
              onChange={(e) => setRouting({ ...routing, bypass_domains: e.target.value })}
            />
            <Text type="secondary" className="block mt-2 text-xs">
              Domains listed here (and their subdomains) bypass the tunnel and route directly over the local internet. Iranian-domestic IPs are detected automatically. Bale domains (<code>.bale.ai</code>, <code>.ble.ir</code>) are always tunneled.
            </Text>
          </Col>
        </Row>
      </Card>

      {/* DNS Speed Benchmark & Auto-Optimizer */}
      <Card
        title={
          <div className="flex items-center gap-2">
            <DashboardOutlined />
            <span>DNS Speed Benchmark &amp; Auto-Optimizer</span>
            {benchmarkStatus?.running && (
              <Tag color="processing" className="animate-pulse">
                SCANNING ({benchmarkStatus.percent}%)
              </Tag>
            )}
          </div>
        }
        bordered={false}
        className="shadow-sm mb-6"
        extra={
          <Space>
            {topOnlineServers.length >= 2 && !benchmarkStatus?.running && (
              <Button
                type="dashed"
                icon={<RocketOutlined />}
                onClick={() => handleApplyDNS(topOnlineServers[0].ip, topOnlineServers[1].ip)}
                title="Automatically configure Primary and Secondary DNS using the top 2 ranked servers"
              >
                ⚡ Auto-Apply Top 2 ({topOnlineServers[0].ip}, {topOnlineServers[1].ip})
              </Button>
            )}
            {benchmarkStatus?.running ? (
              <Button
                danger
                icon={<StopOutlined />}
                onClick={handleStopBenchmark}
              >
                Stop Scan
              </Button>
            ) : (
              <Button
                type="primary"
                icon={<PlayCircleOutlined />}
                loading={benchmarkLoading}
                onClick={handleStartBenchmark}
                className="bg-gradient-to-r from-blue-600 to-indigo-600 hover:from-blue-700 hover:to-indigo-700"
              >
                {benchmarkStatus?.results?.length ? 'Re-scan DNS Servers' : 'Start DNS Benchmark'}
              </Button>
            )}
          </Space>
        }
      >
        <p className="text-secondary text-sm mb-4">
          Benchmarks domestic and international DNS servers by probing <strong>google.com</strong> and all 6 domestic <strong>Bale Meet Gateways</strong> (<code>meet-gwbm1.ble.ir</code> to <code>meet-gwbm6.ble.ir</code>). The ranking algorithm heavily penalizes packet loss and gateway latency spikes to guarantee ultra-responsive Bale WebRTC tunneling.
        </p>

        {benchmarkStatus?.running && (
          <div className="mb-6 p-4 rounded-xl bg-slate-50 dark:bg-slate-800/50 border border-slate-200 dark:border-slate-700">
            <div className="flex justify-between items-center mb-2 text-sm">
              <span className="font-semibold text-slate-700 dark:text-slate-300">
                Scanning DNS servers... [{benchmarkStatus.completed} / {benchmarkStatus.total}]
              </span>
              <span className="text-emerald-600 dark:text-emerald-400 font-medium">
                {benchmarkStatus.online_count} online servers found
              </span>
            </div>
            <Progress
              percent={benchmarkStatus.percent}
              status="active"
              strokeColor={{ from: '#3b82f6', to: '#6366f1' }}
            />
          </div>
        )}

        {/* Filters and Controls */}
        <div className="flex flex-wrap items-center justify-between gap-3 mb-4">
          <Input
            prefix={<SearchOutlined className="text-slate-400" />}
            placeholder="Search by DNS IP..."
            value={dnsSearch}
            onChange={(e) => setDnsSearch(e.target.value)}
            className="w-64"
            allowClear
          />
          <div className="flex items-center gap-4 text-sm text-slate-500">
            <div className="flex items-center gap-2">
              <span>Show online only:</span>
              <Switch checked={onlineOnly} onChange={setOnlineOnly} size="small" />
            </div>
            <span>
              Total Tested: <strong>{benchmarkStatus?.results?.length || 0}</strong>
              {benchmarkStatus?.online_count !== undefined && (
                <> (<strong>{benchmarkStatus.online_count}</strong> active)</>
              )}
            </span>
          </div>
        </div>

        {/* Results Table */}
        <Table
          dataSource={filteredResults}
          columns={dnsColumns}
          rowKey="ip"
          size="middle"
          pagination={{ pageSize: 15, showSizeChanger: true, pageSizeOptions: ['15', '30', '50', '100'] }}
          className="overflow-x-auto"
        />
      </Card>

      {/* Server URLs */}
      {(serverURLs.length > 0 || remoteServerURL) && (
        <Card
          title={<><GlobalOutlined className="mr-2" />Server Addresses</>}
          bordered={false}
          className="shadow-sm mb-6"
        >          <Text type="secondary" className="block mb-3">
            {panelRole === 'SERVER'
              ? 'This server is accessible at the following URLs:'
              : 'Connected remote server addresses:'}
          </Text>
          <Space direction="vertical" className="w-full">
            {serverURLs.map((url: string, idx: number) => (
              <div key={idx} className="flex items-center gap-2 p-2 rounded-lg" style={{ background: 'rgba(99,102,241,0.06)' }}>
                <LinkOutlined className="text-indigo-400" />
                <Text strong className="font-mono text-sm flex-1">{url}</Text>
                <Button
                  type="link"
                  size="small"
                  icon={<LinkOutlined />}
                  onClick={() => window.open(url, '_blank')}
                >
                  Open
                </Button>
              </div>
            ))}
            {remoteServerURL && !serverURLs.includes(remoteServerURL) && (
              <div className="flex items-center gap-2 p-2 rounded-lg" style={{ background: 'rgba(16,185,129,0.06)' }}>
                <LinkOutlined className="text-emerald-400" />
                <Text strong className="font-mono text-sm flex-1">{remoteServerURL}</Text>
                <Tag color="green" className="mr-0">Remote</Tag>
                <Button
                  type="link"
                  size="small"
                  icon={<LinkOutlined />}
                  onClick={() => window.open(remoteServerURL, '_blank')}
                >
                  Open
                </Button>
              </div>
            )}
          </Space>
        </Card>
      )}

      <Card
        title={<><CloudSyncOutlined className="mr-2" />Synchronization</>}
        bordered={false}
        className="shadow-sm mb-6"
        extra={
          panelRole === 'CLIENT' ? (
            <Space>
              <Button type="primary" icon={<SyncOutlined spin={syncing} />} loading={syncing} onClick={handleManualSync}>
                Sync Now
              </Button>
            </Space>
          ) : null
        }
      >
        {panelRole === 'CLIENT' && (
          <>
            <Alert
              message="Automatic Sync Active"
              description="This client admin panel automatically syncs with the server admin panel via long-polling. Account and pairing changes are pushed to the server immediately and pulled from the server in real-time."
              type="success"
              showIcon
              icon={<CheckCircleOutlined />}
              className="mb-4"
            />

            <Card type="inner" title="Push to Server (Restore)" size="small" className="mb-4">
              <Text type="secondary" className="block mb-3">
                If the server lost its data after a redeployment, use these buttons to push your local accounts and pairings to the server to restore them.
              </Text>
              <Space wrap>
                <Button
                  icon={<CloudSyncOutlined />}
                  loading={pushing === 'accounts'}
                  onClick={async () => {
                    setPushing('accounts');
                    try {
                      const res = await api.remoteSyncPushAccounts();
                      message.success(`Pushed ${res.pushed} accounts (${res.failed} failed)`);
                    } catch (e: any) { message.error(e.message); }
                    finally { setPushing(null); }
                  }}
                >
                  Push All Accounts
                </Button>
                <Button
                  icon={<CloudSyncOutlined />}
                  loading={pushing === 'pairings'}
                  onClick={async () => {
                    setPushing('pairings');
                    try {
                      await api.remotePushAllPairings();
                      message.success('All pairings pushed to server');
                    } catch (e: any) { message.error(e.message); }
                    finally { setPushing(null); }
                  }}
                >
                  Push All Pairings
                </Button>
                <Button
                  type="primary"
                  icon={<SyncOutlined />}
                  loading={pushing === 'full'}
                  onClick={async () => {
                    setPushing('full');
                    try {
                      const accRes = await api.remoteSyncPushAccounts();
                      await api.remotePushAllPairings();
                      message.success(`Full restore: ${accRes.pushed} accounts + pairings pushed`);
                    } catch (e: any) { message.error(e.message); }
                    finally { setPushing(null); }
                  }}
                >
                  Full Restore to Server
                </Button>
              </Space>
            </Card>
          </>
        )}

        {panelRole === 'SERVER' && (
          <Alert
            message="Server Admin Panel"
            description="This is the server admin panel. Data is stored locally and synced from client admin panels. If data was lost after a redeployment, use 'Full Restore to Server' from the client admin panel to push accounts and pairings back."
            type="info"
            showIcon
            className="mb-4"
          />
        )}

        {syncStatus ? (
          <Descriptions column={{ xs: 1, md: 2 }} size="small">
            <Descriptions.Item label="Role">
              <Tag color={syncStatus.role === 'server' ? 'blue' : 'green'}>{syncStatus.role?.toUpperCase()}</Tag>
            </Descriptions.Item>
            <Descriptions.Item label="Latest Event ID">
              <Badge status="processing" text={syncStatus.latest_event_id || 0} />
            </Descriptions.Item>
            <Descriptions.Item label="Accounts">{syncStatus.accounts || 0}</Descriptions.Item>
            <Descriptions.Item label="Pairings">{syncStatus.pairings || 0}</Descriptions.Item>
            <Descriptions.Item label="Last Sync Seq">{syncStatus.last_sync_seq || 'Never'}</Descriptions.Item>
            <Descriptions.Item label="Timestamp">{syncStatus.timestamp || '—'}</Descriptions.Item>
          </Descriptions>
        ) : (
          <Text type="secondary">Loading sync status...</Text>
        )}
      </Card>

      {/* Cloud Database Persistence (Clever Cloud Cellar S3) - Active for SERVER */}
      {(panelRole === 'SERVER' || s3Status?.configured) && (
        <Card
          title={<><CloudSyncOutlined className="mr-2 text-indigo-500" />Cloud Database Persistence (Clever Cloud Cellar S3)</>}
          bordered={false}
          className="shadow-sm mb-6"
          extra={
            s3Status?.configured ? (
              <Tag color="success" icon={<CheckCircleOutlined />}>S3 Active & Synced</Tag>
            ) : (
              <Tag color="default">Not Configured</Tag>
            )
          }
        >
          <Alert
            message="Auto-Restore on Startup & Real-Time Sync Active"
            description="When deployed as a Docker container on Clever Cloud, container restarts normally discard local files. With Cellar S3 persistence enabled, the server automatically restores the database from S3 on startup, debounces and syncs every change in real time, and flushes before shutdown."
            type={s3Status?.configured ? "success" : "info"}
            showIcon
            icon={<CloudSyncOutlined />}
            className="mb-4"
          />

          {s3Status?.configured ? (
            <>
              <Descriptions size="small" bordered column={{ xs: 1, sm: 2, md: 3 }} className="mb-4">
                <Descriptions.Item label="S3 Host">
                  <code>{s3Status.host || 'cellar-c2.services.clever-cloud.com'}</code>
                </Descriptions.Item>
                <Descriptions.Item label="Bucket">
                  <Tag color="blue">{s3Status.bucket || 'ble-tunnel-server-db'}</Tag>
                </Descriptions.Item>
                <Descriptions.Item label="Last Sync">
                  <Space size={4}>
                    <Badge status="processing" />
                    <span>{s3Status.last_sync_ago || 'Never'}</span>
                  </Space>
                </Descriptions.Item>
                <Descriptions.Item label="Sync Count">
                  {s3Status.sync_count} syncs completed
                </Descriptions.Item>
                <Descriptions.Item label="Database Size">
                  {s3Status.db_size_bytes ? `${(s3Status.db_size_bytes / 1024).toFixed(1)} KB` : 'N/A'}
                </Descriptions.Item>
                <Descriptions.Item label="Boot Restore">
                  <Tag color={s3Status.restored_init ? 'green' : 'cyan'}>
                    {s3Status.restored_init ? 'Restored on Boot' : 'Ready'}
                  </Tag>
                </Descriptions.Item>
              </Descriptions>

              <Row gutter={[16, 16]}>
                <Col xs={24} md={12}>
                  <Card type="inner" title="Immediate S3 Backup" size="small">
                    <Text type="secondary" className="block mb-3">
                      Forces an immediate WAL checkpoint and uploads the latest database snapshot to Cellar S3.
                    </Text>
                    <Button
                      type="primary"
                      icon={<CloudSyncOutlined />}
                      loading={s3BackupLoading || s3Status.is_syncing}
                      onClick={handleS3BackupNow}
                      block
                    >
                      {s3BackupLoading ? 'Syncing to S3...' : 'Backup to S3 Now'}
                    </Button>
                  </Card>
                </Col>
                <Col xs={24} md={12}>
                  <Card type="inner" title="Restore from S3" size="small">
                    <Text type="secondary" className="block mb-3">
                      Fetches the latest database backup directly from your Clever Cloud Cellar S3 bucket.
                    </Text>
                    <Popconfirm
                      title="Restore from S3?"
                      description="This will overwrite the current database with the latest backup in S3."
                      onConfirm={handleS3RestoreNow}
                      okText="Restore"
                      cancelText="Cancel"
                      okButtonProps={{ danger: true }}
                    >
                      <Button
                        icon={<DownloadOutlined />}
                        loading={s3RestoreLoading}
                        block
                        danger
                      >
                        {s3RestoreLoading ? 'Restoring...' : 'Restore from S3 Now'}
                      </Button>
                    </Popconfirm>
                  </Card>
                </Col>
              </Row>
            </>
          ) : (
            <Alert
              message="Cellar S3 Credentials"
              description="To enable cloud database persistence, set CELLAR_ADDON_HOST, CELLAR_ADDON_KEY_ID, and CELLAR_ADDON_KEY_SECRET environment variables on Clever Cloud."
              type="warning"
              showIcon
            />
          )}
        </Card>
      )}

      <Card
        title={<><DatabaseOutlined className="mr-2" />Backup & Restore</>}
        bordered={false}
        className="shadow-sm mb-6"
      >
        <Alert
          message="Full Database Backup"
          description="Export all accounts (including tokens), pairings, and settings to a JSON file. You can restore this backup on any panel — even after a full redeployment."
          type="info"
          showIcon
          icon={<ExclamationCircleOutlined />}
          className="mb-4"
        />

        <Row gutter={[16, 16]}>
          <Col xs={24} md={12}>
            <Card type="inner" title="Download Backup" size="small">
              <Text type="secondary" className="block mb-3">
                Downloads a JSON file containing all accounts, pairings, and settings.
              </Text>
              <Button
                type="primary"
                icon={<DownloadOutlined />}
                loading={backingUp}
                onClick={async () => {
                  setBackingUp(true);
                  try {
                    const data = await api.dbBackup();
                    const blob = new Blob([JSON.stringify(data, null, 2)], { type: 'application/json' });
                    const url = URL.createObjectURL(blob);
                    const a = document.createElement('a');
                    a.href = url;
                    a.download = `ble-tunnel-backup-${new Date().toISOString().slice(0,10)}.json`;
                    a.click();
                    URL.revokeObjectURL(url);
                    message.success(`Backup downloaded: ${data.accounts?.length || 0} accounts, ${data.pairings?.length || 0} pairings`);
                  } catch (e: any) {
                    message.error(e.message || 'Backup failed');
                  } finally {
                    setBackingUp(false);
                  }
                }}
                block
              >
                Download Backup
              </Button>
            </Card>
          </Col>
          <Col xs={24} md={12}>
            <Card type="inner" title="Restore from Backup" size="small">
              <Text type="secondary" className="block mb-3">
                Upload a backup JSON file to restore accounts, pairings, and settings.
              </Text>
              <input
                ref={fileInputRef}
                type="file"
                accept=".json"
                style={{ display: 'none' }}
                onChange={async (e) => {
                  const file = e.target.files?.[0];
                  if (!file) return;
                  setRestoring(true);
                  try {
                    const text = await file.text();
                    const data = JSON.parse(text);
                    if (!data.version || !data.accounts) {
                      throw new Error('Invalid backup file format');
                    }
                    const res = await api.dbRestore(data);
                    const parts: string[] = [];
                    if (res.accounts_created > 0) parts.push(`${res.accounts_created} accounts created`);
                    if (res.accounts_updated > 0) parts.push(`${res.accounts_updated} accounts updated`);
                    if (res.pairings_created > 0) parts.push(`${res.pairings_created} pairings created`);
                    if (res.settings_restored > 0) parts.push(`${res.settings_restored} settings restored`);
                    if (parts.length > 0) {
                      message.success(`Restore complete: ${parts.join(', ')}`);
                    } else {
                      message.info('Restore complete — all data already up to date');
                    }
                    if (res.accounts_failed > 0 || res.pairings_failed > 0) {
                      message.warning(`${res.accounts_failed} accounts and ${res.pairings_failed} pairings failed`);
                    }
                    loadSyncStatus();
                  } catch (e: any) {
                    message.error(e.message || 'Restore failed');
                  } finally {
                    setRestoring(false);
                    if (fileInputRef.current) fileInputRef.current.value = '';
                  }
                }}
              />
              <Button
                icon={<UploadOutlined />}
                loading={restoring}
                onClick={() => fileInputRef.current?.click()}
                block
                danger
              >
                Upload & Restore
              </Button>
            </Card>
          </Col>
        </Row>

        <Divider className="my-6" />

        <div className="bg-red-500/5 dark:bg-red-500/10 border border-red-200 dark:border-red-900/40 rounded-xl p-4">
          <Row gutter={[16, 16]} align="middle" justify="space-between">
            <Col xs={24} lg={15}>
              <Space direction="vertical" size={2}>
                <Text strong className="text-red-600 dark:text-red-400 text-base">
                  <ExclamationCircleOutlined className="mr-1.5" />
                  Reset Database (Factory Clean)
                </Text>
                <Text type="secondary" className="text-xs block">
                  Permanently clears all accounts, pairings, connection logs, and sync events from the database.
                  <strong> Settings (Bale parameters, DNS, Appearance) and Admin logins are safely preserved.</strong>
                </Text>
              </Space>
            </Col>
            <Col xs={24} lg={9} className="text-right">
              <Space wrap>
                {panelRole === 'CLIENT' && remoteServerURL && (
                  <Popconfirm
                    title="Reset Remote Server Database?"
                    description="This will clear all accounts, pairings, and logs on the Clever Cloud server. Settings and admin credentials will be preserved."
                    onConfirm={handleResetRemoteDatabase}
                    okText="Yes, Reset Server DB"
                    cancelText="Cancel"
                    okButtonProps={{ danger: true }}
                  >
                    <Button danger loading={remoteResetting}>
                      Reset Server DB
                    </Button>
                  </Popconfirm>
                )}
                <Popconfirm
                  title="Reset Database?"
                  description="Permanently clear all accounts, pairings, and logs? Settings and admin logins will be preserved."
                  onConfirm={handleResetDatabase}
                  okText="Yes, Reset Everything"
                  cancelText="Cancel"
                  okButtonProps={{ danger: true }}
                >
                  <Button type="primary" danger loading={resetting}>
                    Reset {panelRole === 'SERVER' ? 'Server' : 'Local'} Database
                  </Button>
                </Popconfirm>
              </Space>
            </Col>
          </Row>
        </div>
      </Card>

      <Card title="Theme Preview" bordered={false} className="shadow-sm">
        <Row gutter={[24, 24]}>
          <Col xs={24} md={8}>
            <Card type="inner" title="Inner Card" size="small">Inner card content</Card>
          </Col>
          <Col xs={24} md={8}>
            <Space direction="vertical" className="w-full">
              <Button type="primary" block>Primary Button</Button>
              <Button block>Default Button</Button>
              <Button danger block>Danger Button</Button>
            </Space>
          </Col>
          <Col xs={24} md={8}>
            <Space>
              <Tag color="success">Active</Tag>
              <Tag color="processing">Processing</Tag>
              <Tag color="error">Error</Tag>
            </Space>
          </Col>
        </Row>
      </Card>
    </div>
  );
}
