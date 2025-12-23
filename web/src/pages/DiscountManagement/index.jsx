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

import React, { useState, useEffect, useMemo } from 'react';
import { useTranslation } from 'react-i18next';
import {
  Table,
  Button,
  Modal,
  Form,
  Input,
  InputNumber,
  Select,
  Space,
  Tag,
  Typography,
  Card,
  Toast,
  Empty,
  Spin,
} from '@douyinfe/semi-ui';
import {
  IconSearch,
  IconRefresh,
  IconEdit,
  IconPercentage,
} from '@douyinfe/semi-icons';
import {
  IllustrationNoResult,
  IllustrationNoResultDark,
} from '@douyinfe/semi-illustrations';
import { API, showError, showSuccess } from '../../helpers';

const { Title, Text } = Typography;

const DiscountManagement = () => {
  const { t } = useTranslation();

  // State
  const [users, setUsers] = useState([]);
  const [loading, setLoading] = useState(false);
  const [selectedRowKeys, setSelectedRowKeys] = useState([]);
  const [activePage, setActivePage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [total, setTotal] = useState(0);
  const [keyword, setKeyword] = useState('');
  const [hasDiscountFilter, setHasDiscountFilter] = useState('');
  const [showBatchModal, setShowBatchModal] = useState(false);
  const [batchDiscountRatio, setBatchDiscountRatio] = useState(1);
  const [submitting, setSubmitting] = useState(false);

  // Quick discount buttons
  const quickDiscountOptions = [
    { label: '90%', value: 0.9 },
    { label: '80%', value: 0.8 },
    { label: '70%', value: 0.7 },
    { label: '60%', value: 0.6 },
    { label: '50%', value: 0.5 },
    { label: t('无优惠'), value: 1 },
  ];

  // Fetch users
  const fetchUsers = async () => {
    setLoading(true);
    try {
      let url = `/api/discount/users?p=${activePage}&size=${pageSize}`;
      if (keyword) {
        url += `&keyword=${encodeURIComponent(keyword)}`;
      }
      if (hasDiscountFilter !== '') {
        url += `&has_discount=${hasDiscountFilter}`;
      }
      const res = await API.get(url);
      const { success, message, data, total: totalCount } = res.data;
      if (success) {
        setUsers(data || []);
        setTotal(totalCount || 0);
      } else {
        showError(message);
      }
    } catch (error) {
      showError(error.message);
    }
    setLoading(false);
  };

  useEffect(() => {
    fetchUsers();
  }, [activePage, pageSize]);

  // Handle search
  const handleSearch = () => {
    setActivePage(1);
    fetchUsers();
  };

  // Handle batch set
  const handleBatchSet = async () => {
    if (selectedRowKeys.length === 0) {
      showError(t('请选择要设置的用户'));
      return;
    }
    if (batchDiscountRatio <= 0 || batchDiscountRatio > 1) {
      showError(t('优惠倍率必须大于0且小于等于1'));
      return;
    }

    setSubmitting(true);
    try {
      const res = await API.post('/api/discount/batch', {
        user_ids: selectedRowKeys,
        discount_ratio: batchDiscountRatio,
      });
      const { success, message } = res.data;
      if (success) {
        showSuccess(message || t('设置成功'));
        setShowBatchModal(false);
        setSelectedRowKeys([]);
        fetchUsers();
      } else {
        showError(message);
      }
    } catch (error) {
      showError(error.message);
    }
    setSubmitting(false);
  };

  // Table columns
  const columns = useMemo(
    () => [
      {
        title: 'ID',
        dataIndex: 'id',
        width: 80,
      },
      {
        title: t('用户名'),
        dataIndex: 'username',
        width: 150,
      },
      {
        title: t('显示名称'),
        dataIndex: 'display_name',
        width: 150,
      },
      {
        title: t('分组'),
        dataIndex: 'group',
        width: 100,
        render: (text) => <Tag color="blue">{text || 'default'}</Tag>,
      },
      {
        title: t('优惠倍率'),
        dataIndex: 'discount_ratio',
        width: 120,
        render: (value) => {
          const ratio = value || 1;
          const discount = Math.round((1 - ratio) * 100);
          if (ratio >= 1) {
            return <Text type="tertiary">{t('无优惠')}</Text>;
          }
          return (
            <Tag color="green" size="large">
              {ratio.toFixed(2)} ({discount}% off)
            </Tag>
          );
        },
      },
    ],
    [t],
  );

  // Row selection config
  const rowSelection = useMemo(
    () => ({
      selectedRowKeys,
      onChange: (keys) => setSelectedRowKeys(keys),
      getCheckboxProps: (record) => ({
        name: record.id,
      }),
    }),
    [selectedRowKeys],
  );

  return (
    <div className="mt-[60px] px-4">
      <Card className="!rounded-2xl shadow-sm border-0 mb-4">
        <div className="flex items-center justify-between mb-4">
          <div className="flex items-center">
            <IconPercentage size="large" className="mr-2 text-blue-500" />
            <div>
              <Title heading={4} className="m-0">
                {t('优惠倍率管理')}
              </Title>
              <Text type="tertiary" className="text-sm">
                {t('管理用户的计费优惠倍率')}
              </Text>
            </div>
          </div>
        </div>

        {/* Filters */}
        <div className="flex flex-wrap gap-3 mb-4">
          <Input
            prefix={<IconSearch />}
            placeholder={t('搜索用户名')}
            value={keyword}
            onChange={setKeyword}
            onEnterPress={handleSearch}
            style={{ width: 200 }}
            showClear
          />
          <Select
            placeholder={t('筛选优惠状态')}
            value={hasDiscountFilter}
            onChange={setHasDiscountFilter}
            style={{ width: 150 }}
            optionList={[
              { label: t('全部'), value: '' },
              { label: t('有优惠'), value: 'true' },
              { label: t('无优惠'), value: 'false' },
            ]}
          />
          <Button icon={<IconSearch />} theme="solid" onClick={handleSearch}>
            {t('搜索')}
          </Button>
          <Button icon={<IconRefresh />} onClick={fetchUsers}>
            {t('刷新')}
          </Button>
        </div>

        {/* Batch actions */}
        {selectedRowKeys.length > 0 && (
          <div className="mb-4 p-3 bg-blue-50 rounded-lg flex items-center justify-between">
            <Text>
              {t('已选择')} <strong>{selectedRowKeys.length}</strong> {t('个用户')}
            </Text>
            <Space>
              {quickDiscountOptions.map((opt) => (
                <Button
                  key={opt.value}
                  size="small"
                  theme={opt.value === 1 ? 'light' : 'solid'}
                  type={opt.value === 1 ? 'tertiary' : 'primary'}
                  onClick={() => {
                    setBatchDiscountRatio(opt.value);
                    setShowBatchModal(true);
                  }}
                >
                  {opt.label}
                </Button>
              ))}
              <Button
                icon={<IconEdit />}
                size="small"
                onClick={() => setShowBatchModal(true)}
              >
                {t('自定义')}
              </Button>
            </Space>
          </div>
        )}

        {/* Table */}
        <Spin spinning={loading}>
          <Table
            columns={columns}
            dataSource={users}
            rowKey="id"
            rowSelection={rowSelection}
            pagination={{
              currentPage: activePage,
              pageSize: pageSize,
              total: total,
              pageSizeOpts: [10, 20, 50, 100],
              showSizeChanger: true,
              onPageSizeChange: (size) => {
                setPageSize(size);
                setActivePage(1);
              },
              onPageChange: setActivePage,
            }}
            empty={
              <Empty
                image={<IllustrationNoResult style={{ width: 150, height: 150 }} />}
                darkModeImage={
                  <IllustrationNoResultDark style={{ width: 150, height: 150 }} />
                }
                description={t('暂无数据')}
              />
            }
          />
        </Spin>
      </Card>

      {/* Batch Set Modal */}
      <Modal
        title={
          <div className="flex items-center">
            <IconPercentage className="mr-2" />
            {t('批量设置优惠倍率')}
          </div>
        }
        visible={showBatchModal}
        onCancel={() => setShowBatchModal(false)}
        onOk={handleBatchSet}
        okText={t('确认设置')}
        cancelText={t('取消')}
        confirmLoading={submitting}
        centered
      >
        <div className="mb-4">
          <Text type="secondary">
            {t('将为选中的')} <strong>{selectedRowKeys.length}</strong> {t('个用户设置优惠倍率')}
          </Text>
        </div>

        <Form.Label>{t('优惠倍率')}</Form.Label>
        <InputNumber
          value={batchDiscountRatio}
          onChange={setBatchDiscountRatio}
          step={0.01}
          min={0.01}
          max={1}
          precision={4}
          style={{ width: '100%' }}
        />
        <Text type="tertiary" className="mt-2 block text-sm">
          {t('范围 0.01 - 1，1 表示无优惠。例如 0.8 表示打 8 折（优惠 20%）')}
        </Text>

        <div className="mt-4">
          <Text className="block mb-2">{t('快捷设置')}</Text>
          <Space wrap>
            {quickDiscountOptions.map((opt) => (
              <Button
                key={opt.value}
                size="small"
                theme={batchDiscountRatio === opt.value ? 'solid' : 'light'}
                onClick={() => setBatchDiscountRatio(opt.value)}
              >
                {opt.label}
              </Button>
            ))}
          </Space>
        </div>
      </Modal>
    </div>
  );
};

export default DiscountManagement;
