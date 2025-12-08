import React, { useState, useEffect, useContext } from 'react';
import { Card, Progress, Tag, Descriptions, Empty, Spin, Divider } from '@douyinfe/semi-ui';
import { API } from '../helpers';
import { showError } from '../helpers/utils';
import { StatusContext } from '../context/Status';

const MySubscriptionsView = () => {
  const [subscriptions, setSubscriptions] = useState([]);
  const [loading, setLoading] = useState(true);
  const [statusState] = useContext(StatusContext);
  const subscriptionPageHTML = statusState?.status?.subscription_page_html || '';

  useEffect(() => {
    fetchData();
  }, []);

  const fetchData = async () => {
    setLoading(true);
    try {
      const res = await API.get('/api/subscription/user/', {
        params: {
          p: 0,
          size: 10,
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
    } else if (dollars >= 0.01) {
      return `$${dollars.toFixed(2)}`;
    } else {
      return `$${dollars.toFixed(4)}`;
    }
  };

  const formatDate = (timestamp) => {
    return new Date(timestamp * 1000).toLocaleString('zh-CN');
  };

  const renderQuotaProgress = (used, limit, label, color = 'blue') => {
    if (limit === 0) {
      return (
        <div style={{ marginBottom: 16 }}>
          <div style={{ marginBottom: 8, fontWeight: 500 }}>
            {label}: <Tag color="green">不限制</Tag>
          </div>
        </div>
      );
    }

    const percent = Math.min((used / limit) * 100, 100);
    const strokeColor = percent > 90 ? 'red' : percent > 70 ? 'orange' : color;

    return (
      <div style={{ marginBottom: 16 }}>
        <div style={{ marginBottom: 8, display: 'flex', justifyContent: 'space-between' }}>
          <span style={{ fontWeight: 500 }}>{label}</span>
          <span>
            {formatQuota(used)} / {formatQuota(limit)}
          </span>
        </div>
        <Progress
          percent={percent}
          stroke={strokeColor}
          showInfo
          format={percent => `${percent.toFixed(1)}%`}
        />
      </div>
    );
  };

  const getStatusTag = (status) => {
    const statusMap = {
      1: { text: '激活中', color: 'green' },
      2: { text: '已过期', color: 'red' },
      3: { text: '已取消', color: 'grey' },
    };
    const config = statusMap[status] || { text: '未知', color: 'grey' };
    return <Tag color={config.color}>{config.text}</Tag>;
  };

  // 渲染可订阅套餐区域
  const renderSubscriptionPlans = () => {
    if (!subscriptionPageHTML) return null;

    return (
      <>
        <Divider style={{ margin: '32px 0' }} />
        <div>
          <h2 style={{ marginBottom: 24 }}>可订阅的套餐</h2>
          <div
            className="subscription-plans-content"
            dangerouslySetInnerHTML={{ __html: subscriptionPageHTML }}
          />
        </div>
      </>
    );
  };

  if (loading) {
    return (
      <div style={{ textAlign: 'center', padding: 64 }}>
        <Spin size="large" />
      </div>
    );
  }

  if (subscriptions.length === 0) {
    return (
      <div>
        <Empty
          title="暂无订阅"
          description="您还没有激活任何订阅套餐"
          style={{ padding: 64 }}
        />
        {renderSubscriptionPlans()}
      </div>
    );
  }

  return (
    <div>
      <h2 style={{ marginBottom: 24 }}>我的订阅</h2>

      {subscriptions.map((sub) => (
        <Card
          key={sub.id}
          title={
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
              <span style={{ fontSize: 18, fontWeight: 600 }}>
                {sub.subscription_info?.name || '未知套餐'}
              </span>
              {getStatusTag(sub.status)}
            </div>
          }
          style={{ marginBottom: 16 }}
          bordered
        >
          {sub.subscription_info?.description && (
            <div style={{ marginBottom: 16, color: 'var(--semi-color-text-1)' }}>
              {sub.subscription_info.description}
            </div>
          )}

          <Descriptions
            row
            size="small"
            style={{ marginBottom: 16 }}
          >
            <Descriptions.Item itemKey="开始时间">
              {formatDate(sub.start_time)}
            </Descriptions.Item>
            <Descriptions.Item itemKey="到期时间">
              <span style={{
                color: sub.status === 1 ? 'var(--semi-color-success)' : 'var(--semi-color-danger)'
              }}>
                {formatDate(sub.expire_time)}
              </span>
            </Descriptions.Item>
            <Descriptions.Item itemKey="有效期">
              {sub.subscription_info?.duration_days} 天
            </Descriptions.Item>
          </Descriptions>

          {sub.status === 1 && sub.subscription_info && (
            <div style={{
              padding: 16,
              backgroundColor: 'var(--semi-color-fill-0)',
              borderRadius: 8,
            }}>
              <h4 style={{ marginBottom: 16 }}>额度使用情况</h4>

              {renderQuotaProgress(
                sub.daily_quota_used,
                sub.subscription_info.daily_quota_limit,
                '今日额度',
                'blue'
              )}

              {renderQuotaProgress(
                sub.weekly_quota_used,
                sub.subscription_info.weekly_quota_limit,
                '本周额度',
                'cyan'
              )}

              {renderQuotaProgress(
                sub.monthly_quota_used,
                sub.subscription_info.monthly_quota_limit,
                '本月额度',
                'violet'
              )}

              <div style={{
                marginTop: 16,
                padding: 12,
                backgroundColor: 'var(--semi-color-info-light-default)',
                borderRadius: 4,
                fontSize: 12,
                color: 'var(--semi-color-text-2)'
              }}>
                <div>• 每日额度在每天 00:00 自动重置</div>
                <div>• 每周额度在每周一 00:00 自动重置</div>
                <div>• 每月额度在每月 1 号 00:00 自动重置</div>
              </div>
            </div>
          )}

          {sub.status === 2 && (
            <div style={{
              padding: 16,
              backgroundColor: 'var(--semi-color-danger-light-default)',
              borderRadius: 8,
              color: 'var(--semi-color-danger)',
            }}>
              此订阅已过期，请续费或激活新的订阅套餐
            </div>
          )}
        </Card>
      ))}

      {renderSubscriptionPlans()}
    </div>
  );
};

export default MySubscriptionsView;
