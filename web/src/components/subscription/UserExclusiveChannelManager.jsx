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

import React, { useEffect, useState } from 'react';
import {
  Modal,
  Button,
  Table,
  Tag,
  Space,
  Select,
  Empty,
  Spin,
  Typography,
  Popconfirm,
  Banner,
} from '@douyinfe/semi-ui';
import { IconPlus, IconDelete, IconRefresh, IconLink } from '@douyinfe/semi-icons';
import { useTranslation } from 'react-i18next';
import { API, showError, showSuccess } from '../../helpers';
import { getChannelIcon } from '../../helpers/render';

const { Text, Title } = Typography;

const UserExclusiveChannelManager = ({ visible, onClose, userId, userName }) => {
  const { t } = useTranslation();
  const [loading, setLoading] = useState(false);
  const [channels, setChannels] = useState([]);
  const [availableChannels, setAvailableChannels] = useState([]);
  const [selectedChannelId, setSelectedChannelId] = useState(null);
  const [adding, setAdding] = useState(false);

  // Load user's exclusive channels
  const loadExclusiveChannels = async () => {
    if (!userId) return;
    setLoading(true);
    try {
      const res = await API.get(`/api/subscription/exclusive/user/${userId}/channels`);
      if (res.data.success) {
        // 提取渠道信息，添加 channel_id 用于后续操作
        const channelList = (res.data.data || []).map(item => ({
          ...item.channel_info,
          _bindingId: item.id,
          _channelId: item.channel_id,
        })).filter(c => c && c.id);
        setChannels(channelList);
      } else {
        showError(res.data.message || t('获取失败'));
      }
    } catch (error) {
      const errMsg = error.response?.data?.message || error.message || t('请求失败');
      showError(t('获取专属渠道失败') + ': ' + errMsg);
    } finally {
      setLoading(false);
    }
  };

  // Load available channels for binding
  const loadAvailableChannels = async () => {
    try {
      const res = await API.get('/api/subscription/exclusive/available_channels');
      if (res.data.success) {
        // Filter out channels already bound to this user
        const existingIds = new Set(channels.map(c => c.id));
        const available = (res.data.data || []).filter(c => !existingIds.has(c.id));
        setAvailableChannels(available);
      } else {
        showError(res.data.message || t('获取失败'));
      }
    } catch (error) {
      const errMsg = error.response?.data?.message || error.message || t('请求失败');
      showError(t('获取可用渠道失败') + ': ' + errMsg);
    }
  };

  useEffect(() => {
    if (visible && userId) {
      loadExclusiveChannels();
    }
  }, [visible, userId]);

  useEffect(() => {
    if (visible && channels) {
      loadAvailableChannels();
    }
  }, [visible, channels]);

  // Add channel to user's exclusive group
  const handleAddChannel = async () => {
    if (!selectedChannelId) {
      showError(t('请选择要添加的渠道'));
      return;
    }
    setAdding(true);
    try {
      const res = await API.post(`/api/subscription/exclusive/user/${userId}/channel`, {
        channel_id: selectedChannelId,
      });
      if (res.data.success) {
        showSuccess(t('添加专属渠道成功'));
        setSelectedChannelId(null);
        loadExclusiveChannels();
      } else {
        showError(res.data.message || t('添加失败'));
      }
    } catch (error) {
      const errMsg = error.response?.data?.message || error.message || t('请求失败');
      showError(t('添加专属渠道失败') + ': ' + errMsg);
    } finally {
      setAdding(false);
    }
  };

  // Remove channel from user's exclusive group
  const handleRemoveChannel = async (channelId) => {
    try {
      const res = await API.delete(`/api/subscription/exclusive/user/${userId}/channel/${channelId}`);
      if (res.data.success) {
        showSuccess(t('移除专属渠道成功'));
        loadExclusiveChannels();
      } else {
        showError(res.data.message || t('移除失败'));
      }
    } catch (error) {
      const errMsg = error.response?.data?.message || error.message || t('请求失败');
      showError(t('移除专属渠道失败') + ': ' + errMsg);
    }
  };

  const columns = [
    {
      title: 'ID',
      dataIndex: 'id',
      width: 80,
    },
    {
      title: t('渠道名称'),
      dataIndex: 'name',
      width: 180,
      render: (text, record) => (
        <Space>
          {getChannelIcon(record.type)}
          <Text>{text}</Text>
        </Space>
      ),
    },
    {
      title: t('类型'),
      dataIndex: 'type',
      width: 100,
      render: (type) => {
        const typeMap = {
          1: 'OpenAI',
          14: 'Claude',
          24: 'Gemini',
          43: 'DeepSeek',
          // Add more type mappings as needed
        };
        return typeMap[type] || `类型 ${type}`;
      },
    },
    {
      title: t('支持模型'),
      dataIndex: 'models',
      width: 250,
      render: (models) => {
        if (!models) return '-';
        const modelList = models.split(',').slice(0, 3);
        const hasMore = models.split(',').length > 3;
        return (
          <div style={{ display: 'flex', flexWrap: 'wrap', gap: '4px' }}>
            {modelList.map((model, idx) => (
              <Tag key={idx} size="small" color="blue">
                {model.trim()}
              </Tag>
            ))}
            {hasMore && <Tag size="small">+{models.split(',').length - 3}</Tag>}
          </div>
        );
      },
    },
    {
      title: t('状态'),
      dataIndex: 'status',
      width: 80,
      render: (status) => (
        <Tag color={status === 1 ? 'green' : 'red'}>
          {status === 1 ? t('启用') : t('禁用')}
        </Tag>
      ),
    },
    {
      title: t('操作'),
      width: 100,
      fixed: 'right',
      render: (_, record) => (
        <Popconfirm
          title={t('确认移除')}
          content={t('确定要从该用户的专属渠道中移除此渠道吗？')}
          onConfirm={() => handleRemoveChannel(record.id)}
        >
          <Button
            size="small"
            type="danger"
            theme="light"
            icon={<IconDelete />}
          >
            {t('移除')}
          </Button>
        </Popconfirm>
      ),
    },
  ];

  const renderChannelOption = (item) => {
    if (!item) return null;
    const icon = getChannelIcon(item.type);
    const modelCount = item.models ? item.models.split(',').length : 0;
    return (
      <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
        {icon}
        <span>{item.name || '-'}</span>
        <Tag size="small" color="blue">{modelCount} {t('模型')}</Tag>
      </div>
    );
  };

  return (
    <Modal
      title={
        <Space>
          <IconLink />
          <span>{t('管理用户专属渠道')}</span>
          {userName && <Tag color="blue">{userName}</Tag>}
        </Space>
      }
      visible={visible}
      onCancel={onClose}
      footer={
        <Button onClick={onClose}>{t('关闭')}</Button>
      }
      width={900}
      style={{ maxWidth: '95vw' }}
    >
      <Spin spinning={loading}>
        <Banner
          type="info"
          description={t('专属渠道仅该用户可用，使用专属分组时仅消耗订阅额度。')}
          style={{ marginBottom: 16 }}
        />

        {/* Add channel section */}
        <div style={{
          display: 'flex',
          gap: '12px',
          marginBottom: 16,
          padding: '16px',
          backgroundColor: 'var(--semi-color-fill-0)',
          borderRadius: '8px',
        }}>
          <Select
            placeholder={t('选择要添加的渠道')}
            style={{ flex: 1 }}
            value={selectedChannelId}
            onChange={setSelectedChannelId}
            optionList={availableChannels.map(c => ({
              value: c.id,
              label: c.name,
              type: c.type,
              models: c.models,
            }))}
            renderSelectedItem={(option) => option?.label || ''}
            renderOptionItem={(renderProps) => {
              const { disabled, selected, label, value, focused, onMouseEnter, onClick, ...rest } = renderProps;
              const item = availableChannels.find(c => c.id === value);
              return (
                <div
                  style={{
                    padding: '8px 12px',
                    cursor: disabled ? 'not-allowed' : 'pointer',
                    backgroundColor: selected ? 'var(--semi-color-primary-light-default)' : (focused ? 'var(--semi-color-fill-0)' : 'transparent'),
                  }}
                  onMouseEnter={onMouseEnter}
                  onClick={onClick}
                >
                  {item ? renderChannelOption(item) : label}
                </div>
              );
            }}
            filter
            showClear
          />
          <Button
            theme="solid"
            type="primary"
            icon={<IconPlus />}
            loading={adding}
            onClick={handleAddChannel}
            disabled={!selectedChannelId}
          >
            {t('添加渠道')}
          </Button>
          <Button
            icon={<IconRefresh />}
            onClick={() => {
              loadExclusiveChannels();
              loadAvailableChannels();
            }}
          >
            {t('刷新')}
          </Button>
        </div>

        {/* Channels table */}
        {channels.length > 0 ? (
          <Table
            columns={columns}
            dataSource={channels}
            rowKey="id"
            pagination={false}
            size="small"
          />
        ) : (
          <Empty
            image={<IconLink style={{ fontSize: 48, color: 'var(--semi-color-text-2)' }} />}
            title={t('暂无专属渠道')}
            description={t('请从上方选择渠道添加到该用户的专属分组')}
          />
        )}

        {/* Info section */}
        <div style={{
          marginTop: 16,
          padding: '12px',
          backgroundColor: 'var(--semi-color-fill-0)',
          borderRadius: '8px',
        }}>
          <Text type="tertiary" size="small">
            {t('专属分组名称')}: <Text code>sub_user_{userId}</Text>
          </Text>
        </div>
      </Spin>
    </Modal>
  );
};

export default UserExclusiveChannelManager;
