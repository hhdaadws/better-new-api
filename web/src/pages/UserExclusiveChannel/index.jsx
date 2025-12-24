import React, { useEffect, useState } from 'react';
import {
  Table,
  Button,
  Card,
  Space,
  Tag,
  Typography,
  Empty,
} from '@douyinfe/semi-ui';
import { IconRefresh, IconLink } from '@douyinfe/semi-icons';
import { useTranslation } from 'react-i18next';
import { API, showError } from '../../helpers';
import UserExclusiveChannelManager from '../../components/subscription/UserExclusiveChannelManager';

const { Title, Text } = Typography;

const UserExclusiveChannel = () => {
  const { t } = useTranslation();
  const [loading, setLoading] = useState(false);
  const [users, setUsers] = useState([]);
  const [selectedUser, setSelectedUser] = useState(null);
  const [managerVisible, setManagerVisible] = useState(false);

  const loadUsers = async () => {
    setLoading(true);
    try {
      const res = await API.get('/api/subscription/exclusive/users');
      if (res.data.success) {
        setUsers(res.data.data || []);
      } else {
        showError(res.data.message);
      }
    } catch (error) {
      showError(t('获取用户列表失败') + ': ' + error.message);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadUsers();
  }, []);

  const handleManage = (user) => {
    setSelectedUser(user);
    setManagerVisible(true);
  };

  const handleManagerClose = () => {
    setManagerVisible(false);
    setSelectedUser(null);
    loadUsers();
  };

  const formatTime = (timestamp) => {
    if (!timestamp || timestamp === 0) {
      return t('永久');
    }
    const date = new Date(timestamp * 1000);
    return date.toLocaleDateString('zh-CN', {
      year: 'numeric',
      month: '2-digit',
      day: '2-digit',
    });
  };

  const columns = [
    {
      title: t('用户ID'),
      dataIndex: 'user_id',
      width: 80,
    },
    {
      title: t('用户名'),
      dataIndex: 'username',
      width: 120,
    },
    {
      title: t('显示名'),
      dataIndex: 'display_name',
      width: 120,
      render: (text) => text || '-',
    },
    {
      title: t('邮箱'),
      dataIndex: 'email',
      width: 180,
      render: (text) => text || '-',
    },
    {
      title: t('订阅套餐'),
      dataIndex: 'subscription_name',
      width: 120,
      render: (text) => (
        <Tag color="blue">{text}</Tag>
      ),
    },
    {
      title: t('已配置渠道'),
      dataIndex: 'channel_count',
      width: 100,
      render: (count) => (
        <Tag color={count > 0 ? 'green' : 'grey'}>
          {count} {t('个')}
        </Tag>
      ),
    },
    {
      title: t('到期时间'),
      dataIndex: 'expire_time',
      width: 120,
      render: (time) => formatTime(time),
    },
    {
      title: t('操作'),
      width: 100,
      fixed: 'right',
      render: (_, record) => (
        <Button
          theme="solid"
          type="primary"
          size="small"
          icon={<IconLink />}
          onClick={() => handleManage(record)}
        >
          {t('管理')}
        </Button>
      ),
    },
  ];

  return (
    <div className="mt-[60px] px-2">
      <Card>
        <div style={{
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'center',
          marginBottom: 16
        }}>
          <Title heading={5} style={{ margin: 0 }}>
            <Space>
              <IconLink />
              {t('专属渠道管理')}
            </Space>
          </Title>
          <Button
            icon={<IconRefresh />}
            onClick={loadUsers}
            loading={loading}
          >
            {t('刷新')}
          </Button>
        </div>

        <Text type="tertiary" style={{ display: 'block', marginBottom: 16 }}>
          {t('管理有专属分组权限的用户的专属渠道配置。只有订阅了启用专属分组的套餐的用户才会显示在此列表中。')}
        </Text>

        {users.length > 0 ? (
          <Table
            columns={columns}
            dataSource={users}
            rowKey="user_id"
            loading={loading}
            pagination={{
              pageSize: 20,
              showTotal: true,
            }}
            scroll={{ x: 900 }}
          />
        ) : (
          <Empty
            image={<IconLink style={{ fontSize: 48, color: 'var(--semi-color-text-2)' }} />}
            title={t('暂无数据')}
            description={t('当前没有订阅了启用专属分组套餐的用户')}
          />
        )}
      </Card>

      <UserExclusiveChannelManager
        visible={managerVisible}
        onClose={handleManagerClose}
        userId={selectedUser?.user_id}
        userName={selectedUser?.username}
      />
    </div>
  );
};

export default UserExclusiveChannel;
