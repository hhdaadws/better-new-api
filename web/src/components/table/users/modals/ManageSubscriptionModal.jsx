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
  SideSheet,
  Table,
  Button,
  Space,
  Tag,
  Form,
  Progress,
  Typography,
  Spin,
  Empty,
  Popconfirm,
  DatePicker,
  InputNumber,
  Select,
  Banner,
} from '@douyinfe/semi-ui';
import { IconPlus, IconEdit, IconDelete, IconLink } from '@douyinfe/semi-icons';
import { API, showError, showSuccess, renderQuota, timestamp2string } from '../../../../helpers';
import { useIsMobile } from '../../../../hooks/common/useIsMobile';
import UserExclusiveChannelManager from '../../../subscription/UserExclusiveChannelManager';

const { Text } = Typography;

const ManageSubscriptionModal = ({ visible, onCancel, user, t, refresh }) => {
  const isMobile = useIsMobile();
  const [loading, setLoading] = useState(false);
  const [subscriptions, setSubscriptions] = useState([]);
  const [allSubscriptions, setAllSubscriptions] = useState([]);
  const [showAddModal, setShowAddModal] = useState(false);
  const [showEditModal, setShowEditModal] = useState(false);
  const [editingSubscription, setEditingSubscription] = useState(null);
  const [addForm, setAddForm] = useState({ subscription_id: null, duration_days: null });
  const [editForm, setEditForm] = useState({ subscription_id: null, expire_time: null });
  const [showExclusiveChannels, setShowExclusiveChannels] = useState(false);
  const [hasExclusiveSubscription, setHasExclusiveSubscription] = useState(false);

  // Load user's subscriptions
  const loadUserSubscriptions = async () => {
    if (!user?.id) return;
    setLoading(true);
    try {
      const res = await API.get(`/api/user/${user.id}/subscriptions?p=0&size=100`);
      if (res.data.success) {
        // API returns paginated data: { items: [...], total: ... }
        const data = res.data.data;
        const subs = Array.isArray(data) ? data : (data?.items || []);
        setSubscriptions(subs);

        // Check if any active subscription has exclusive group enabled
        const hasExclusive = subs.some(s =>
          s.status === 1 && s.subscription_info?.enable_exclusive_group
        );
        setHasExclusiveSubscription(hasExclusive);
      } else {
        showError(res.data.message);
      }
    } catch (e) {
      showError(e.message);
    }
    setLoading(false);
  };

  // Load all available subscription packages
  const loadAllSubscriptions = async () => {
    try {
      const res = await API.get('/api/subscription/');
      if (res.data.success) {
        // API returns paginated data: { items: [...], total: ... }
        const data = res.data.data;
        setAllSubscriptions(Array.isArray(data) ? data : (data?.items || []));
      }
    } catch (e) {
      showError(e.message);
    }
  };

  useEffect(() => {
    if (visible && user?.id) {
      loadUserSubscriptions();
      loadAllSubscriptions();
    }
  }, [visible, user?.id]);

  // Render subscription status
  const renderStatus = (status) => {
    switch (status) {
      case 1:
        return <Tag color="green">{t('激活')}</Tag>;
      case 2:
        return <Tag color="grey">{t('过期')}</Tag>;
      case 3:
        return <Tag color="red">{t('已取消')}</Tag>;
      case 4:
        return <Tag color="orange">{t('已替换')}</Tag>;
      default:
        return <Tag color="grey">{t('未知')}</Tag>;
    }
  };

  // Render quota usage
  const renderUsage = (record) => {
    const items = [];
    const info = record.subscription_info;
    if (!info) return '-';

    const addUsageItem = (label, used, limit) => {
      if (limit > 0) {
        const percent = Math.min((used / limit) * 100, 100);
        items.push(
          <div key={label} className="mb-1">
            <Text size="small">{label}: {renderQuota(used)} / {renderQuota(limit)}</Text>
            <Progress percent={percent} size="small" style={{ width: 120 }} />
          </div>
        );
      }
    };

    addUsageItem(t('日用量'), record.daily_quota_used || 0, info.daily_quota_limit || 0);
    addUsageItem(t('周用量'), record.weekly_quota_used || 0, info.weekly_quota_limit || 0);
    addUsageItem(t('总用量'), record.total_quota_used || 0, info.total_quota_limit || 0);

    return items.length > 0 ? <div>{items}</div> : '-';
  };

  // Add subscription
  const handleAddSubscription = async () => {
    if (!addForm.subscription_id) {
      showError(t('请选择订阅套餐'));
      return;
    }
    setLoading(true);
    try {
      const payload = { subscription_id: addForm.subscription_id };
      if (addForm.duration_days && addForm.duration_days > 0) {
        payload.duration_days = addForm.duration_days;
      }
      const res = await API.post(`/api/user/${user.id}/subscription`, payload);
      if (res.data.success) {
        showSuccess(t('订阅添加成功'));
        setShowAddModal(false);
        setAddForm({ subscription_id: null, duration_days: null });
        loadUserSubscriptions();
        if (refresh) refresh();
      } else {
        showError(res.data.message);
      }
    } catch (e) {
      showError(e.message);
    }
    setLoading(false);
  };

  // Update subscription
  const handleUpdateSubscription = async () => {
    if (!editForm.subscription_id && !editForm.expire_time) {
      showError(t('请至少修改一项'));
      return;
    }
    setLoading(true);
    try {
      const payload = {};
      if (editForm.subscription_id) {
        payload.subscription_id = editForm.subscription_id;
      }
      if (editForm.expire_time) {
        payload.expire_time = Math.floor(editForm.expire_time.getTime() / 1000);
      }
      const res = await API.put(`/api/user/${user.id}/subscription/${editingSubscription.id}`, payload);
      if (res.data.success) {
        showSuccess(t('订阅修改成功'));
        setShowEditModal(false);
        setEditingSubscription(null);
        setEditForm({ subscription_id: null, expire_time: null });
        loadUserSubscriptions();
        if (refresh) refresh();
      } else {
        showError(res.data.message);
      }
    } catch (e) {
      showError(e.message);
    }
    setLoading(false);
  };

  // Cancel subscription
  const handleCancelSubscription = async (subscriptionId) => {
    setLoading(true);
    try {
      const res = await API.delete(`/api/user/${user.id}/subscription/${subscriptionId}`);
      if (res.data.success) {
        showSuccess(t('订阅已取消'));
        loadUserSubscriptions();
        if (refresh) refresh();
      } else {
        showError(res.data.message);
      }
    } catch (e) {
      showError(e.message);
    }
    setLoading(false);
  };

  // Open edit modal
  const openEditModal = (record) => {
    setEditingSubscription(record);
    setEditForm({
      subscription_id: null,
      expire_time: record.expire_time ? new Date(record.expire_time * 1000) : null,
    });
    setShowEditModal(true);
  };

  // Table columns
  const columns = [
    {
      title: 'ID',
      dataIndex: 'id',
      width: 60,
    },
    {
      title: t('套餐名称'),
      dataIndex: 'subscription_info',
      width: 120,
      render: (info) => info?.name || '-',
    },
    {
      title: t('状态'),
      dataIndex: 'status',
      width: 80,
      render: (status) => renderStatus(status),
    },
    {
      title: t('用量'),
      dataIndex: 'usage',
      width: 180,
      render: (_, record) => renderUsage(record),
    },
    {
      title: t('有效期'),
      dataIndex: 'expire_time',
      width: 150,
      render: (time, record) => (
        <div className="text-xs">
          <div>{t('开始')}: {timestamp2string(record.start_time)}</div>
          <div>{t('结束')}: {timestamp2string(time)}</div>
        </div>
      ),
    },
    {
      title: t('操作'),
      dataIndex: 'operate',
      width: 120,
      render: (_, record) => {
        if (record.status !== 1) return '-';
        return (
          <Space>
            <Button
              size="small"
              type="tertiary"
              icon={<IconEdit />}
              onClick={() => openEditModal(record)}
            />
            <Popconfirm
              title={t('确定要取消此订阅吗？')}
              onConfirm={() => handleCancelSubscription(record.id)}
            >
              <Button size="small" type="danger" icon={<IconDelete />} />
            </Popconfirm>
          </Space>
        );
      },
    },
  ];

  // Filter enabled subscriptions for selection
  const enabledSubscriptions = allSubscriptions.filter((s) => s.status === 1);

  const content = (
    <Spin spinning={loading}>
      {/* Exclusive Group Banner */}
      {hasExclusiveSubscription && (
        <Banner
          type="info"
          description={
            <div className="flex items-center justify-between">
              <span>{t('该用户有启用专属分组的订阅套餐，可以配置专属渠道')}</span>
              <Button
                size="small"
                theme="solid"
                icon={<IconLink />}
                onClick={() => setShowExclusiveChannels(true)}
              >
                {t('管理专属渠道')}
              </Button>
            </div>
          }
          style={{ marginBottom: 16 }}
        />
      )}

      <div className="mb-4">
        <Space>
          <Button
            icon={<IconPlus />}
            theme="solid"
            onClick={() => setShowAddModal(true)}
          >
            {t('添加订阅')}
          </Button>
          {hasExclusiveSubscription && (
            <Button
              icon={<IconLink />}
              theme="light"
              onClick={() => setShowExclusiveChannels(true)}
            >
              {t('管理专属渠道')}
            </Button>
          )}
        </Space>
      </div>

      <Table
        columns={columns}
        dataSource={subscriptions}
        pagination={false}
        size="small"
        empty={
          <Empty description={t('暂无订阅记录')} />
        }
      />

      {/* Add Subscription Modal */}
      <Modal
        title={t('添加订阅')}
        visible={showAddModal}
        onOk={handleAddSubscription}
        onCancel={() => {
          setShowAddModal(false);
          setAddForm({ subscription_id: null, duration_days: null });
        }}
        okText={t('确定')}
        cancelText={t('取消')}
      >
        <Form labelPosition="top">
          <Form.Slot label={t('选择套餐')}>
            <Select
              placeholder={t('请选择订阅套餐')}
              value={addForm.subscription_id}
              onChange={(value) => setAddForm({ ...addForm, subscription_id: value })}
              style={{ width: '100%' }}
              optionList={enabledSubscriptions.map((s) => ({
                value: s.id,
                label: `${s.name} - ${renderQuota(s.daily_quota_limit)}/日`,
              }))}
            />
          </Form.Slot>
          <Form.Slot label={t('有效天数（可选，留空使用套餐默认值）')}>
            <InputNumber
              placeholder={t('天数')}
              value={addForm.duration_days}
              onChange={(value) => setAddForm({ ...addForm, duration_days: value })}
              min={1}
              style={{ width: '100%' }}
            />
          </Form.Slot>
        </Form>
      </Modal>

      {/* Edit Subscription Modal */}
      <Modal
        title={t('修改订阅')}
        visible={showEditModal}
        onOk={handleUpdateSubscription}
        onCancel={() => {
          setShowEditModal(false);
          setEditingSubscription(null);
          setEditForm({ subscription_id: null, expire_time: null });
        }}
        okText={t('确定')}
        cancelText={t('取消')}
      >
        <Form labelPosition="top">
          <Form.Slot label={t('更换套餐（可选）')}>
            <Select
              placeholder={t('保持不变')}
              value={editForm.subscription_id}
              onChange={(value) => setEditForm({ ...editForm, subscription_id: value })}
              style={{ width: '100%' }}
              showClear
              optionList={enabledSubscriptions.map((s) => ({
                value: s.id,
                label: `${s.name} - ${renderQuota(s.daily_quota_limit)}/日`,
              }))}
            />
          </Form.Slot>
          <Form.Slot label={t('修改过期时间（可选）')}>
            <DatePicker
              type="dateTime"
              placeholder={t('选择新的过期时间')}
              value={editForm.expire_time}
              onChange={(value) => setEditForm({ ...editForm, expire_time: value })}
              style={{ width: '100%' }}
            />
          </Form.Slot>
        </Form>
      </Modal>

      {/* Exclusive Channels Manager */}
      <UserExclusiveChannelManager
        visible={showExclusiveChannels}
        onClose={() => setShowExclusiveChannels(false)}
        userId={user?.id}
        userName={user?.username}
      />
    </Spin>
  );

  const title = (
    <Space>
      <Tag color="blue" shape="circle">{t('订阅')}</Tag>
      <Text strong>{t('管理订阅')} - {user?.username}</Text>
    </Space>
  );

  if (isMobile) {
    return (
      <SideSheet
        title={title}
        visible={visible}
        onCancel={onCancel}
        placement="bottom"
        height="90%"
      >
        {content}
      </SideSheet>
    );
  }

  return (
    <Modal
      title={title}
      visible={visible}
      onCancel={onCancel}
      footer={null}
      width={900}
      style={{ maxHeight: '80vh' }}
      bodyStyle={{ overflow: 'auto' }}
    >
      {content}
    </Modal>
  );
};

export default ManageSubscriptionModal;
