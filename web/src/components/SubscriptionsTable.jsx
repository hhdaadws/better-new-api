import React, { useState, useEffect } from 'react';
import { Button, Table, Tag, Space, Modal, Form, Input, InputNumber, Select, Popconfirm, TextArea, Switch } from '@douyinfe/semi-ui';
import { API } from '../helpers';
import { showError, showSuccess } from '../helpers/utils';

const SubscriptionsTable = () => {
  const [subscriptions, setSubscriptions] = useState([]);
  const [loading, setLoading] = useState(false);
  const [modalVisible, setModalVisible] = useState(false);
  const [redemptionModalVisible, setRedemptionModalVisible] = useState(false);
  const [editingId, setEditingId] = useState(null);
  const [currentSubscription, setCurrentSubscription] = useState(null);
  const [redemptionKeys, setRedemptionKeys] = useState([]);
  const [groupOptions, setGroupOptions] = useState([]);
  const formApi = React.useRef();
  const redemptionFormApi = React.useRef();

  const columns = [
    {
      title: 'ID',
      dataIndex: 'id',
      width: 80,
    },
    {
      title: '套餐名称',
      dataIndex: 'name',
      width: 150,
    },
    {
      title: '描述',
      dataIndex: 'description',
      width: 200,
      render: (text) => (
        <div style={{ wordBreak: 'break-word' }}>{text || '-'}</div>
      ),
    },
    {
      title: '每日限额',
      dataIndex: 'daily_quota_limit',
      width: 120,
      render: (value) => value === 0 ? <Tag color="green">不限制</Tag> : formatQuota(value),
    },
    {
      title: '每周限额',
      dataIndex: 'weekly_quota_limit',
      width: 120,
      render: (value) => value === 0 ? <Tag color="green">不限制</Tag> : formatQuota(value),
    },
    {
      title: '总限额',
      dataIndex: 'total_quota_limit',
      width: 120,
      render: (value) => formatQuota(value),
    },
    {
      title: '有效期',
      dataIndex: 'duration_days',
      width: 100,
      render: (value) => `${value} 天`,
    },
    {
      title: '允许分组',
      dataIndex: 'allowed_groups',
      width: 200,
      render: (value) => {
        try {
          const groups = JSON.parse(value);
          return (
            <div>
              {groups.map(g => <Tag key={g} style={{ marginBottom: 4 }}>{g}</Tag>)}
            </div>
          );
        } catch {
          return value;
        }
      },
    },
    {
      title: '状态',
      dataIndex: 'status',
      width: 100,
      render: (status) => (
        <Tag color={status === 1 ? 'green' : 'red'}>
          {status === 1 ? '启用' : '禁用'}
        </Tag>
      ),
    },
    {
      title: '专属分组',
      dataIndex: 'enable_exclusive_group',
      width: 100,
      render: (value) => (
        <Tag color={value ? 'blue' : 'grey'}>
          {value ? '已启用' : '未启用'}
        </Tag>
      ),
    },
    {
      title: '操作',
      width: 280,
      fixed: 'right',
      render: (record) => (
        <Space>
          <Button size="small" onClick={() => handleEdit(record)}>编辑</Button>
          <Popconfirm
            title="确认删除"
            content="确定要删除此套餐吗？"
            onConfirm={() => handleDelete(record.id)}
          >
            <Button size="small" type="danger">删除</Button>
          </Popconfirm>
          <Button
            size="small"
            theme="solid"
            type="tertiary"
            onClick={() => handleCreateRedemption(record)}
          >
            生成兑换码
          </Button>
        </Space>
      ),
    },
  ];

  useEffect(() => {
    fetchData();
    fetchGroups();
  }, []);

  const fetchGroups = async () => {
    try {
      const res = await API.get('/api/group/');
      if (res.data.success) {
        setGroupOptions(
          res.data.data.map((group) => ({
            label: group,
            value: group,
          }))
        );
      }
    } catch (error) {
      console.error('获取分组失败:', error);
    }
  };

  const fetchData = async () => {
    setLoading(true);
    try {
      const res = await API.get('/api/subscription/', {
        params: {
          p: 0,
          size: 100,
        }
      });
      if (res.data.success) {
        setSubscriptions(res.data.data?.items || []);
      } else {
        showError(res.data.message);
      }
    } catch (error) {
      showError('获取数据失败: ' + error.message);
    } finally {
      setLoading(false);
    }
  };

  // QuotaPerUnit = 500000 = $1
  const QUOTA_PER_UNIT = 500000;

  const formatQuota = (quota) => {
    // 将内部额度单位转换为美元显示
    const dollars = quota / QUOTA_PER_UNIT;
    if (dollars >= 1) {
      return `$${dollars.toFixed(2)}`;
    } else {
      // 小于1美元时显示更多小数位
      return `$${dollars.toFixed(4)}`;
    }
  };

  // 将美元转换为内部额度单位
  const dollarsToQuota = (dollars) => {
    return Math.round(dollars * QUOTA_PER_UNIT);
  };

  // 将内部额度单位转换为美元
  const quotaToDollars = (quota) => {
    return quota / QUOTA_PER_UNIT;
  };

  const handleAdd = () => {
    setEditingId(null);
    setModalVisible(true);
    formApi.current?.reset();
  };

  const handleEdit = (record) => {
    setEditingId(record.id);
    formApi.current?.setValues({
      ...record,
      allowed_groups: JSON.parse(record.allowed_groups),
      // 将内部额度单位转换为美元用于表单显示
      daily_quota_limit: quotaToDollars(record.daily_quota_limit),
      weekly_quota_limit: quotaToDollars(record.weekly_quota_limit),
      total_quota_limit: quotaToDollars(record.total_quota_limit),
    });
    setModalVisible(true);
  };

  const handleDelete = async (id) => {
    try {
      const res = await API.delete(`/api/subscription/${id}`);
      if (res.data.success) {
        showSuccess('删除成功');
        fetchData();
      } else {
        showError(res.data.message);
      }
    } catch (error) {
      showError('删除失败: ' + error.message);
    }
  };

  const handleSubmit = async (values) => {
    try {
      values.allowed_groups = JSON.stringify(values.allowed_groups);
      // 将美元转换为内部额度单位
      values.daily_quota_limit = dollarsToQuota(values.daily_quota_limit || 0);
      values.weekly_quota_limit = dollarsToQuota(values.weekly_quota_limit || 0);
      values.total_quota_limit = dollarsToQuota(values.total_quota_limit || 0);

      if (editingId) {
        values.id = editingId;
        const res = await API.put(`/api/subscription/${editingId}`, values);
        if (res.data.success) {
          showSuccess('更新成功');
          setModalVisible(false);
          fetchData();
        } else {
          showError(res.data.message);
        }
      } else {
        const res = await API.post('/api/subscription/', values);
        if (res.data.success) {
          showSuccess('创建成功');
          setModalVisible(false);
          fetchData();
        } else {
          showError(res.data.message);
        }
      }
    } catch (error) {
      showError('操作失败: ' + error.message);
    }
  };

  const handleCreateRedemption = (record) => {
    setCurrentSubscription(record);
    setRedemptionKeys([]);
    setRedemptionModalVisible(true);
    redemptionFormApi.current?.reset();
  };

  const handleGenerateRedemption = async (values) => {
    try {
      const res = await API.post('/api/subscription/redemption', {
        name: values.name,
        subscription_id: currentSubscription.id,
        count: values.count,
        expired_time: values.expired_time ? Math.floor(values.expired_time.getTime() / 1000) : 0,
      });

      if (res.data.success) {
        setRedemptionKeys(res.data.data);
        showSuccess(`成功生成 ${res.data.data.length} 个兑换码`);
      } else {
        showError(res.data.message);
      }
    } catch (error) {
      showError('生成失败: ' + error.message);
    }
  };

  const copyToClipboard = (text) => {
    navigator.clipboard.writeText(text).then(() => {
      showSuccess('已复制到剪贴板');
    }).catch(() => {
      showError('复制失败');
    });
  };

  return (
    <div>
      <div style={{ marginBottom: 16 }}>
        <Button onClick={handleAdd} theme="solid" type="primary">
          添加订阅套餐
        </Button>
      </div>

      <Table
        columns={columns}
        dataSource={subscriptions}
        loading={loading}
        rowKey="id"
        pagination={{
          pageSize: 10,
          showSizeChanger: true,
        }}
      />

      {/* 添加/编辑套餐模态框 */}
      <Modal
        title={editingId ? '编辑订阅套餐' : '添加订阅套餐'}
        visible={modalVisible}
        onCancel={() => setModalVisible(false)}
        footer={null}
        style={{ width: 600 }}
      >
        <Form
          getFormApi={(api) => (formApi.current = api)}
          onSubmit={handleSubmit}
          labelPosition="left"
          labelWidth={120}
        >
          <Form.Input
            field="name"
            label="套餐名称"
            rules={[{ required: true, message: '请输入套餐名称' }]}
            placeholder="例如：基础月卡"
          />
          <Form.TextArea
            field="description"
            label="套餐描述"
            placeholder="简要描述套餐的特点"
            rows={3}
          />
          <Form.InputNumber
            field="daily_quota_limit"
            label="每日限额"
            initValue={0}
            min={0}
            step={0.1}
            precision={2}
            prefix="$"
            placeholder="0表示不限制"
            extraText="输入美元金额，0表示不限制每日使用"
            style={{ width: '100%' }}
          />
          <Form.InputNumber
            field="weekly_quota_limit"
            label="每周限额"
            initValue={0}
            min={0}
            step={0.1}
            precision={2}
            prefix="$"
            placeholder="0表示不限制"
            extraText="输入美元金额，0表示不限制每周使用"
            style={{ width: '100%' }}
          />
          <Form.InputNumber
            field="total_quota_limit"
            label="总限额"
            rules={[{ required: true, message: '请输入总限额' }]}
            min={0.01}
            step={1}
            precision={2}
            prefix="$"
            extraText="输入美元金额，这是套餐的总费用额度（订阅期内不重置）"
            style={{ width: '100%' }}
          />
          <Form.InputNumber
            field="duration_days"
            label="有效期（天）"
            initValue={30}
            min={1}
            suffix="天"
            style={{ width: '100%' }}
          />
          <Form.Select
            field="allowed_groups"
            label="允许的分组"
            multiple
            filter
            rules={[{ required: true, message: '请选择至少一个分组' }]}
            placeholder="选择可以使用此套餐的分组"
            optionList={groupOptions}
            style={{ width: '100%' }}
          />
          <Form.Select
            field="status"
            label="状态"
            initValue={1}
            style={{ width: '100%' }}
          >
            <Select.Option value={1}>启用</Select.Option>
            <Select.Option value={2}>禁用</Select.Option>
          </Form.Select>
          <Form.Switch
            field="enable_exclusive_group"
            label="启用专属分组"
            initValue={false}
            extraText="启用后，管理员可为每个用户配置专属渠道，用户使用专属分组时仅消耗订阅额度"
          />

          <div style={{ display: 'flex', justifyContent: 'flex-end', marginTop: 16 }}>
            <Space>
              <Button onClick={() => setModalVisible(false)}>取消</Button>
              <Button htmlType="submit" theme="solid" type="primary">
                提交
              </Button>
            </Space>
          </div>
        </Form>
      </Modal>

      {/* 生成兑换码模态框 */}
      <Modal
        title={`生成 ${currentSubscription?.name} 兑换码`}
        visible={redemptionModalVisible}
        onCancel={() => setRedemptionModalVisible(false)}
        footer={null}
        style={{ width: 600 }}
      >
        <Form
          getFormApi={(api) => (redemptionFormApi.current = api)}
          onSubmit={handleGenerateRedemption}
          labelPosition="left"
          labelWidth={120}
        >
          <Form.Input
            field="name"
            label="兑换码名称"
            rules={[{ required: true, message: '请输入兑换码名称' }]}
            placeholder="例如：2025年1月活动"
          />
          <Form.InputNumber
            field="count"
            label="生成数量"
            initValue={1}
            min={1}
            max={100}
            rules={[{ required: true, message: '请输入生成数量' }]}
            style={{ width: '100%' }}
          />
          <Form.DatePicker
            field="expired_time"
            label="过期时间"
            type="dateTime"
            placeholder="不设置则永久有效"
            style={{ width: '100%' }}
          />

          <div style={{ display: 'flex', justifyContent: 'flex-end', marginTop: 16 }}>
            <Button htmlType="submit" theme="solid" type="primary">
              生成兑换码
            </Button>
          </div>
        </Form>

        {redemptionKeys.length > 0 && (
          <div style={{ marginTop: 24 }}>
            <h4>生成的兑换码：</h4>
            <div style={{
              maxHeight: 300,
              overflowY: 'auto',
              border: '1px solid var(--semi-color-border)',
              borderRadius: 4,
              padding: 12,
              backgroundColor: 'var(--semi-color-fill-0)'
            }}>
              {redemptionKeys.map((key, index) => (
                <div key={index} style={{
                  marginBottom: 8,
                  padding: 8,
                  backgroundColor: 'var(--semi-color-bg-2)',
                  borderRadius: 4,
                  display: 'flex',
                  justifyContent: 'space-between',
                  alignItems: 'center'
                }}>
                  <code style={{ flex: 1, fontSize: 12 }}>{key}</code>
                  <Button
                    size="small"
                    onClick={() => copyToClipboard(key)}
                    style={{ marginLeft: 8 }}
                  >
                    复制
                  </Button>
                </div>
              ))}
            </div>
            <Button
              style={{ marginTop: 12, width: '100%' }}
              onClick={() => copyToClipboard(redemptionKeys.join('\n'))}
              theme="solid"
            >
              复制全部
            </Button>
          </div>
        )}
      </Modal>
    </div>
  );
};

export default SubscriptionsTable;
