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
import { useTranslation } from 'react-i18next';
import {
  Modal,
  SideSheet,
  Table,
  Button,
  Space,
  Tag,
  Form,
  Select,
  InputNumber,
  Popconfirm,
  Typography,
  Progress,
  DatePicker,
} from '@douyinfe/semi-ui';
import { IconPlus, IconEdit, IconDelete } from '@douyinfe/semi-icons';
import { API, showError, showSuccess, renderQuota, timestamp2string } from '../../../../helpers';
import { useIsMobile } from '../../../../hooks/common/useIsMobile';

const { Text } = Typography;

const ManageSubscriptionModal = ({ visible, handleClose, user, refresh }) => {
  const { t } = useTranslation();
  const isMobile = useIsMobile();
  const [loading, setLoading] = useState(false);
  const [subscriptions, setSubscriptions] = useState([]);
  const [allSubscriptions, setAllSubscriptions] = useState([]);
  const [showAddModal, setShowAddModal] = useState(false);
  const [showEditModal, setShowEditModal] = useState(false);
  const [editingSubscription, setEditingSubscription] = useState(null);

  useEffect(() => {
    if (visible && user) {
      loadUserSubscriptions();
      loadAllSubscriptions();
    }
  }, [visible, user]);

  const loadUserSubscriptions = async () => {
    setLoading(true);
    try {
      const res = await API.get(`/api/user/${user.id}/subscriptions?p=0&size=100`);
      const { success, message, data } = res.data;
      if (success) {
        setSubscriptions(data.items || []);
      } else {
        showError(message);
      }
    } catch (error) {
      showError(error.message);
    } finally {
      setLoading(false);
    }
  };

  const loadAllSubscriptions = async () => {
    try {
      const res = await API.get('/api/subscription/?p=0&size=100');
      const { success, message, data } = res.data;
      if (success) {
        setAllSubscriptions(data.items || []);
      }
    } catch (error) {
      showError(error.message);
    }
  };

  const handleAddSubscription = async (values) => {
    try {
      const res = await API.post(`/api/user/${user.id}/subscription`, values);
      const { success, message } = res.data;
      if (success) {
        showSuccess(t('订阅添加成功'));
        setShowAddModal(false);
        loadUserSubscriptions();
        refresh();
      } else {
        showError(message);
      }
    } catch (error) {
      showError(error.message);
    }
  };

  const handleUpdateSubscription = async (values) => {
    try {
      const payload = {};
      if (values.subscription_id) {
        payload.subscription_id = values.subscription_id;
      }
      if (values.expire_time) {
        payload.expire_time = Math.floor(values.expire_time.getTime() / 1000);
      }

      const res = await API.put(
        `/api/user/${user.id}/subscription/${editingSubscription.id}`,
        payload
      );
      const { success, message } = res.data;
      if (success) {
        showSuccess(t('订阅修改成功'));
        setShowEditModal(false);
        setEditingSubscription(null);
        loadUserSubscriptions();
        refresh();
      } else {
        showError(message);
      }
    } catch (error) {
      showError(error.message);
    }
  };

  const handleCancelSubscription = async (subId) => {
    try {
      const res = await API.delete(`/api/user/${user.id}/subscription/${subId}`);
      const { success, message } = res.data;
      if (success) {
        showSuccess(t('订阅已取消'));
        loadUserSubscriptions();
        refresh();
      } else {
        showError(message);
      }
    } catch (error) {
      showError(error.message);
    }
  };

  const renderStatus = (status) => {
    const statusMap = {
      1: { text: t('激活'), color: 'green' },
      2: { text: t('已过期'), color: 'grey' },
      3: { text: t('已取消'), color: 'red' },
      4: { text: t('已替换'), color: 'orange' },
    };
    const statusInfo = statusMap[status] || { text: t('未知'), color: 'grey' };
    return <Tag color={statusInfo.color}>{statusInfo.text}</Tag>;
  };

  const renderUsage = (record) => {
    if (!record.subscription_info) return '-';

    const daily = record.daily_quota_used || 0;
    const dailyLimit = record.subscription_info.daily_quota_limit || 0;
    const weekly = record.weekly_quota_used || 0;
    const weeklyLimit = record.subscription_info.weekly_quota_limit || 0;
    const total = record.total_quota_used || 0;
    const totalLimit = record.subscription_info.total_quota_limit || 0;

    return (
      <Space vertical align='start'>
        {dailyLimit > 0 && (
          <div className='w-full'>
            <Text size='small'>
              {t('日')}: {renderQuota(daily)} / {renderQuota(dailyLimit)}
            </Text>
            <Progress
              percent={(daily / dailyLimit) * 100}
              size='small'
              showInfo={false}
            />
          </div>
        )}
        {weeklyLimit > 0 && (
          <div className='w-full'>
            <Text size='small'>
              {t('周')}: {renderQuota(weekly)} / {renderQuota(weeklyLimit)}
            </Text>
            <Progress
              percent={(weekly / weeklyLimit) * 100}
              size='small'
              showInfo={false}
            />
          </div>
        )}
        {totalLimit > 0 && (
          <div className='w-full'>
            <Text size='small'>
              {t('总')}: {renderQuota(total)} / {renderQuota(totalLimit)}
            </Text>
            <Progress
              percent={(total / totalLimit) * 100}
              size='small'
              showInfo={false}
            />
          </div>
        )}
      </Space>
    );
  };

  const columns = [
    {
      title: 'ID',
      dataIndex: 'id',
      width: 80,
    },
    {
      title: t('套餐名称'),
      dataIndex: 'subscription_info',
      render: (info) => info?.name || '-',
    },
    {
      title: t('状态'),
      dataIndex: 'status',
      render: (status) => renderStatus(status),
    },
    {
      title: t('用量'),
      key: 'usage',
      render: (text, record) => renderUsage(record),
    },
    {
      title: t('有效期'),
      key: 'time',
      render: (text, record) => (
        <Space vertical align='start'>
          <Text size='small'>{timestamp2string(record.start_time)}</Text>
          <Text size='small'>~ {timestamp2string(record.expire_time)}</Text>
        </Space>
      ),
    },
    {
      title: t('操作'),
      key: 'actions',
      fixed: 'right',
      width: 150,
      render: (text, record) => {
        if (record.status !== 1) return null;
        return (
          <Space>
            <Button
              size='small'
              type='tertiary'
              icon={<IconEdit />}
              onClick={() => {
                setEditingSubscription(record);
                setShowEditModal(true);
              }}
            >
              {t('修改')}
            </Button>
            <Popconfirm
              title={t('确定要取消此订阅吗？')}
              content={t('取消后用户将无法继续使用此订阅')}
              onConfirm={() => handleCancelSubscription(record.id)}
            >
              <Button size='small' type='danger' icon={<IconDelete />}>
                {t('取消')}
              </Button>
            </Popconfirm>
          </Space>
        );
      },
    },
  ];

  const AddSubscriptionForm = () => (
    <Form onSubmit={handleAddSubscription}>
      <Form.Select
        field='subscription_id'
        label={t('选择套餐')}
        placeholder={t('请选择套餐')}
        rules={[{ required: true, message: t('请选择套餐') }]}
      >
        {allSubscriptions
          .filter((s) => s.status === 1)
          .map((sub) => (
            <Form.Select.Option key={sub.id} value={sub.id}>
              {sub.name} ({t('限额')}: {renderQuota(sub.total_quota_limit)}, {t('时长')}: {sub.duration_days}{t('天')})
            </Form.Select.Option>
          ))}
      </Form.Select>
      <Form.InputNumber
        field='duration_days'
        label={t('有效天数')}
        placeholder={t('留空使用套餐默认时长')}
        min={1}
        max={3650}
      />
      <Space className='w-full justify-end mt-4'>
        <Button onClick={() => setShowAddModal(false)}>{t('取消')}</Button>
        <Button type='primary' htmlType='submit'>
          {t('添加')}
        </Button>
      </Space>
    </Form>
  );

  const EditSubscriptionForm = () => {
    if (!editingSubscription) return null;

    return (
      <Form
        onSubmit={handleUpdateSubscription}
        initValues={{
          expire_time: new Date(editingSubscription.expire_time * 1000),
        }}
      >
        <Form.Select
          field='subscription_id'
          label={t('更换套餐')}
          placeholder={t('留空保持不变')}
        >
          {allSubscriptions
            .filter((s) => s.status === 1)
            .map((sub) => (
              <Form.Select.Option key={sub.id} value={sub.id}>
                {sub.name} ({t('限额')}: {renderQuota(sub.total_quota_limit)})
              </Form.Select.Option>
            ))}
        </Form.Select>
        <Form.DatePicker
          field='expire_time'
          label={t('过期时间')}
          type='dateTime'
          format='yyyy-MM-dd HH:mm:ss'
        />
        <Space className='w-full justify-end mt-4'>
          <Button
            onClick={() => {
              setShowEditModal(false);
              setEditingSubscription(null);
            }}
          >
            {t('取消')}
          </Button>
          <Button type='primary' htmlType='submit'>
            {t('保存')}
          </Button>
        </Space>
      </Form>
    );
  };

  const content = (
    <div className='p-4'>
      <Space vertical align='start' className='w-full' spacing='medium'>
        <div className='flex justify-between items-center w-full'>
          <Text strong>
            {t('用户')}: {user?.username} (ID: {user?.id})
          </Text>
          <Button
            type='primary'
            icon={<IconPlus />}
            onClick={() => setShowAddModal(true)}
          >
            {t('添加订阅')}
          </Button>
        </div>

        <Table
          columns={columns}
          dataSource={subscriptions}
          loading={loading}
          pagination={false}
          size='small'
        />
      </Space>

      <Modal
        title={t('添加订阅')}
        visible={showAddModal}
        onCancel={() => setShowAddModal(false)}
        footer={null}
      >
        <AddSubscriptionForm />
      </Modal>

      <Modal
        title={t('修改订阅')}
        visible={showEditModal}
        onCancel={() => {
          setShowEditModal(false);
          setEditingSubscription(null);
        }}
        footer={null}
      >
        <EditSubscriptionForm />
      </Modal>
    </div>
  );

  return isMobile ? (
    <SideSheet
      title={t('管理用户订阅')}
      visible={visible}
      onCancel={handleClose}
      width='90%'
    >
      {content}
    </SideSheet>
  ) : (
    <Modal
      title={t('管理用户订阅')}
      visible={visible}
      onCancel={handleClose}
      footer={null}
      style={{ width: 900 }}
    >
      {content}
    </Modal>
  );
};

export default ManageSubscriptionModal;
