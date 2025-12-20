/*
Copyright (C) 2025 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/

import React, { useState, useEffect } from 'react';
import { useTranslation } from 'react-i18next';
import {
  Modal,
  Button,
  Table,
  Tag,
  Typography,
  Space,
  Tooltip,
  Popconfirm,
  Empty,
  Spin,
  Row,
  Col,
  Badge,
  Progress,
  Card,
} from '@douyinfe/semi-ui';
import {
  IllustrationNoResult,
  IllustrationNoResultDark,
} from '@douyinfe/semi-illustrations';
import {
  API,
  showError,
  showSuccess,
  timestamp2string,
  copy,
} from '../../../../helpers';

const { Text } = Typography;

const StickySessionModal = ({ visible, onCancel, channel, onRefresh }) => {
  const { t } = useTranslation();
  const [loading, setLoading] = useState(false);
  const [sessionInfo, setSessionInfo] = useState(null);
  const [operationLoading, setOperationLoading] = useState({});

  // Load sticky session data
  const loadStickySessionInfo = async () => {
    if (!channel?.id) return;

    setLoading(true);
    try {
      const res = await API.get(`/api/channel/${channel.id}/sticky_sessions`);

      if (res.data.success) {
        setSessionInfo(res.data.data);
      } else {
        showError(res.data.message);
      }
    } catch (error) {
      console.error(error);
      showError(t('获取粘性会话信息失败'));
    } finally {
      setLoading(false);
    }
  };

  // Release a specific session
  const handleReleaseSession = async (sessionHash) => {
    const operationId = `release_${sessionHash}`;
    setOperationLoading((prev) => ({ ...prev, [operationId]: true }));

    try {
      const res = await API.delete(
        `/api/channel/${channel.id}/sticky_sessions/${sessionHash}`,
      );

      if (res.data.success) {
        showSuccess(t('会话已释放'));
        await loadStickySessionInfo();
        onRefresh && onRefresh();
      } else {
        showError(res.data.message);
      }
    } catch (error) {
      showError(t('释放会话失败'));
    } finally {
      setOperationLoading((prev) => ({ ...prev, [operationId]: false }));
    }
  };

  // Release all sessions
  const handleReleaseAll = async () => {
    setOperationLoading((prev) => ({ ...prev, release_all: true }));

    try {
      const res = await API.delete(
        `/api/channel/${channel.id}/sticky_sessions`,
      );

      if (res.data.success) {
        showSuccess(res.data.message || t('所有会话已释放'));
        await loadStickySessionInfo();
        onRefresh && onRefresh();
      } else {
        showError(res.data.message);
      }
    } catch (error) {
      showError(t('释放所有会话失败'));
    } finally {
      setOperationLoading((prev) => ({ ...prev, release_all: false }));
    }
  };

  // Copy session hash
  const handleCopyHash = async (hash) => {
    const ok = await copy(hash);
    if (ok) {
      showSuccess(t('已复制会话哈希'));
    } else {
      showError(t('复制失败'));
    }
  };

  // Effect to load data when modal opens
  useEffect(() => {
    if (visible && channel?.id) {
      loadStickySessionInfo();
    }
  }, [visible, channel?.id]);

  // Reset when modal closes
  useEffect(() => {
    if (!visible) {
      setSessionInfo(null);
    }
  }, [visible]);

  // Calculate progress
  const sessionCount = sessionInfo?.session_count || 0;
  const maxCount = sessionInfo?.max_count || 0;
  const usagePercent =
    maxCount > 0 ? Math.min(Math.round((sessionCount / maxCount) * 100), 100) : 0;

  // Daily bind info
  const dailyBindLimit = sessionInfo?.daily_bind_limit || 0;
  const dailyBindCount = sessionInfo?.daily_bind_count || 0;
  const dailyBindRemaining = dailyBindLimit > 0 ? Math.max(dailyBindLimit - dailyBindCount, 0) : -1;
  const dailyBindPercent =
    dailyBindLimit > 0 ? Math.min(Math.round((dailyBindCount / dailyBindLimit) * 100), 100) : 0;

  // Format TTL display
  const formatTTL = (seconds) => {
    if (seconds <= 0) return t('已过期');
    if (seconds < 60) return `${seconds}${t('秒')}`;
    if (seconds < 3600) return `${Math.floor(seconds / 60)}${t('分钟')}`;
    return `${Math.floor(seconds / 3600)}${t('小时')} ${Math.floor((seconds % 3600) / 60)}${t('分钟')}`;
  };

  // Table columns definition
  const columns = [
    {
      title: t('会话哈希'),
      dataIndex: 'session_hash',
      render: (text) => (
        <Tooltip content={text}>
          <Text
            code
            style={{ fontSize: '12px', cursor: 'pointer' }}
            onClick={() => handleCopyHash(text)}
          >
            {text.length > 16 ? `${text.substring(0, 16)}...` : text}
          </Text>
        </Tooltip>
      ),
    },
    {
      title: t('分组'),
      dataIndex: 'group',
      render: (text) => (
        <Tag size='small' color='blue'>
          {text}
        </Tag>
      ),
    },
    {
      title: t('模型'),
      dataIndex: 'model',
      render: (text) => (
        <Tooltip content={text}>
          <Text style={{ maxWidth: '150px', display: 'block' }} ellipsis>
            {text}
          </Text>
        </Tooltip>
      ),
    },
    {
      title: t('用户名'),
      dataIndex: 'username',
      render: (text) => {
        if (!text) return <Text type='quaternary'>-</Text>;
        return (
          <Tag size='small' color='cyan'>
            {text}
          </Tag>
        );
      },
    },
    {
      title: t('令牌'),
      dataIndex: 'token_name',
      render: (text) => {
        if (!text) return <Text type='quaternary'>-</Text>;
        return (
          <Tooltip content={text}>
            <Text code style={{ fontSize: '12px' }}>
              {text.length > 12 ? `${text.substring(0, 12)}...` : text}
            </Text>
          </Tooltip>
        );
      },
    },
    {
      title: t('创建时间'),
      dataIndex: 'created_at',
      render: (time) => {
        if (!time) return <Text type='quaternary'>-</Text>;
        return (
          <Tooltip content={timestamp2string(time)}>
            <Text style={{ fontSize: '12px' }}>{timestamp2string(time)}</Text>
          </Tooltip>
        );
      },
    },
    {
      title: t('剩余时间'),
      dataIndex: 'ttl',
      render: (ttl) => {
        if (ttl <= 0) {
          return (
            <Tag color='red' size='small'>
              {t('已过期')}
            </Tag>
          );
        }
        return (
          <Tag color='green' size='small'>
            {formatTTL(ttl)}
          </Tag>
        );
      },
    },
    {
      title: t('操作'),
      key: 'action',
      fixed: 'right',
      width: 100,
      render: (_, record) => (
        <Popconfirm
          title={t('确定要释放此会话吗？')}
          content={t('释放后该会话将重新分配渠道')}
          onConfirm={() => handleReleaseSession(record.session_hash)}
          position={'topRight'}
        >
          <Button
            type='danger'
            size='small'
            loading={operationLoading[`release_${record.session_hash}`]}
          >
            {t('释放')}
          </Button>
        </Popconfirm>
      ),
    },
  ];

  return (
    <Modal
      title={
        <Space>
          <Text>{t('粘性会话管理')}</Text>
          {channel?.name && (
            <Tag size='small' shape='circle' color='white'>
              {channel.name}
            </Tag>
          )}
        </Space>
      }
      visible={visible}
      onCancel={onCancel}
      width={900}
      footer={null}
    >
      <div className='flex flex-col mb-5'>
        {/* Stats */}
        <div
          className='rounded-xl p-4 mb-3'
          style={{
            background: 'var(--semi-color-bg-1)',
            border: '1px solid var(--semi-color-border)',
          }}
        >
          <Row gutter={16} align='middle'>
            <Col span={8}>
              <div
                style={{
                  background: 'var(--semi-color-bg-0)',
                  border: '1px solid var(--semi-color-border)',
                  borderRadius: 12,
                  padding: 12,
                }}
              >
                <div className='flex items-center gap-2 mb-2'>
                  <Badge dot type='primary' />
                  <Text type='tertiary'>{t('当前会话数')}</Text>
                </div>
                <div className='flex items-end gap-2 mb-2'>
                  <Text
                    style={{ fontSize: 24, fontWeight: 700, color: '#3b82f6' }}
                  >
                    {sessionCount}
                  </Text>
                  {maxCount > 0 && (
                    <Text
                      style={{ fontSize: 18, color: 'var(--semi-color-text-2)' }}
                    >
                      / {maxCount}
                    </Text>
                  )}
                  {maxCount === 0 && (
                    <Text
                      style={{ fontSize: 14, color: 'var(--semi-color-text-2)' }}
                    >
                      ({t('无限制')})
                    </Text>
                  )}
                </div>
                {maxCount > 0 && (
                  <Progress
                    percent={usagePercent}
                    showInfo={false}
                    size='small'
                    stroke={usagePercent >= 90 ? '#ef4444' : usagePercent >= 70 ? '#f59e0b' : '#3b82f6'}
                    style={{ height: 6, borderRadius: 999 }}
                  />
                )}
              </div>
            </Col>
            <Col span={8}>
              <div
                style={{
                  background: 'var(--semi-color-bg-0)',
                  border: '1px solid var(--semi-color-border)',
                  borderRadius: 12,
                  padding: 12,
                }}
              >
                <div className='flex items-center gap-2 mb-2'>
                  <Badge dot type='success' />
                  <Text type='tertiary'>{t('会话上限')}</Text>
                </div>
                <div className='flex items-end gap-2'>
                  <Text
                    style={{ fontSize: 24, fontWeight: 700, color: '#22c55e' }}
                  >
                    {maxCount === 0 ? t('无限制') : maxCount}
                  </Text>
                </div>
              </div>
            </Col>
            <Col span={8}>
              <div
                style={{
                  background: 'var(--semi-color-bg-0)',
                  border: '1px solid var(--semi-color-border)',
                  borderRadius: 12,
                  padding: 12,
                }}
              >
                <div className='flex items-center gap-2 mb-2'>
                  <Badge dot type='warning' />
                  <Text type='tertiary'>{t('过期时间')}</Text>
                </div>
                <div className='flex items-end gap-2'>
                  <Text
                    style={{ fontSize: 24, fontWeight: 700, color: '#f59e0b' }}
                  >
                    {sessionInfo?.ttl_minutes || 60}
                  </Text>
                  <Text
                    style={{ fontSize: 14, color: 'var(--semi-color-text-2)' }}
                  >
                    {t('分钟')}
                  </Text>
                </div>
              </div>
            </Col>
          </Row>
          {/* Daily Bind Limit - only show if configured */}
          {dailyBindLimit > 0 && (
            <Row gutter={16} align='middle' style={{ marginTop: 12 }}>
              <Col span={24}>
                <div
                  style={{
                    background: 'var(--semi-color-bg-0)',
                    border: '1px solid var(--semi-color-border)',
                    borderRadius: 12,
                    padding: 12,
                  }}
                >
                  <div className='flex items-center gap-2 mb-2'>
                    <Badge dot style={{ backgroundColor: '#8b5cf6' }} />
                    <Text type='tertiary'>{t('今日绑定额度')}</Text>
                  </div>
                  <div className='flex items-center gap-4'>
                    <div className='flex items-end gap-2'>
                      <Text
                        style={{ fontSize: 24, fontWeight: 700, color: '#8b5cf6' }}
                      >
                        {dailyBindCount}
                      </Text>
                      <Text
                        style={{ fontSize: 18, color: 'var(--semi-color-text-2)' }}
                      >
                        / {dailyBindLimit}
                      </Text>
                    </div>
                    <div style={{ flex: 1, maxWidth: 200 }}>
                      <Progress
                        percent={dailyBindPercent}
                        showInfo={false}
                        size='small'
                        stroke={dailyBindPercent >= 100 ? '#ef4444' : dailyBindPercent >= 80 ? '#f59e0b' : '#8b5cf6'}
                        style={{ height: 6, borderRadius: 999 }}
                      />
                    </div>
                    <Text
                      style={{
                        fontSize: 14,
                        color: dailyBindRemaining === 0 ? '#ef4444' : 'var(--semi-color-text-2)',
                      }}
                    >
                      {dailyBindRemaining === 0
                        ? t('已用尽')
                        : t('剩余 {{count}}', { count: dailyBindRemaining })}
                    </Text>
                  </div>
                </div>
              </Col>
            </Row>
          )}
        </div>

        {/* Table */}
        <div className='flex-1 flex flex-col min-h-0'>
          <Spin spinning={loading}>
            <Card className='!rounded-xl'>
              <Table
                title={() => (
                  <Row gutter={12} style={{ width: '100%' }}>
                    <Col span={12}>
                      <Text type='tertiary'>
                        {t('共 {{count}} 个活跃会话', { count: sessionCount })}
                      </Text>
                    </Col>
                    <Col
                      span={12}
                      style={{ display: 'flex', justifyContent: 'flex-end' }}
                    >
                      <Space>
                        <Button
                          size='small'
                          type='tertiary'
                          onClick={loadStickySessionInfo}
                          loading={loading}
                        >
                          {t('刷新')}
                        </Button>
                        {sessionCount > 0 && (
                          <Popconfirm
                            title={t('确定要释放所有会话吗？')}
                            content={t('此操作将释放该渠道的所有粘性会话绑定')}
                            onConfirm={handleReleaseAll}
                            okType={'danger'}
                            position={'topRight'}
                          >
                            <Button
                              size='small'
                              type='danger'
                              loading={operationLoading.release_all}
                            >
                              {t('释放全部')}
                            </Button>
                          </Popconfirm>
                        )}
                      </Space>
                    </Col>
                  </Row>
                )}
                columns={columns}
                dataSource={sessionInfo?.sessions || []}
                pagination={false}
                size='small'
                bordered={false}
                rowKey='session_hash'
                scroll={{ x: 'max-content' }}
                empty={
                  <Empty
                    image={
                      <IllustrationNoResult
                        style={{ width: 140, height: 140 }}
                      />
                    }
                    darkModeImage={
                      <IllustrationNoResultDark
                        style={{ width: 140, height: 140 }}
                      />
                    }
                    title={t('暂无粘性会话')}
                    description={
                      sessionInfo?.session_count === 0 && sessionInfo?.max_count === 0
                        ? t('该渠道未启用粘性会话功能')
                        : t('该渠道当前没有绑定的会话')
                    }
                    style={{ padding: 30 }}
                  />
                }
              />
            </Card>
          </Spin>
        </div>
      </div>
    </Modal>
  );
};

export default StickySessionModal;
