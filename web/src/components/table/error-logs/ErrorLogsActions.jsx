/*
Copyright (C) 2025 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
*/

import React from 'react';
import { Button, Typography, Space, Switch, Popconfirm } from '@douyinfe/semi-ui';
import { IconSetting, IconRefresh, IconDelete, IconDeleteStroked } from '@douyinfe/semi-icons';

const { Title, Text } = Typography;

const ErrorLogsActions = (logsData) => {
  const {
    t,
    logCount,
    loading,
    refresh,
    setShowColumnSelector,
    compactMode,
    setCompactMode,
    selectedRowKeys,
    deleteLogs,
    clearAllErrorLogs,
    isAdminUser,
  } = logsData;

  return (
    <div className='flex flex-col sm:flex-row justify-between items-start sm:items-center gap-4'>
      <div className='flex items-center gap-3'>
        <Title heading={4} style={{ margin: 0 }}>
          {t('错误日志')}
        </Title>
        <Text type='tertiary'>
          {t('共 {{count}} 条', { count: logCount })}
        </Text>
      </div>
      <Space>
        <div className='flex items-center gap-2'>
          <Text type='tertiary'>{t('紧凑模式')}</Text>
          <Switch
            checked={compactMode}
            onChange={setCompactMode}
            size='small'
          />
        </div>
        <Button
          icon={<IconSetting />}
          onClick={() => setShowColumnSelector(true)}
        >
          {t('列设置')}
        </Button>
        {isAdminUser && selectedRowKeys && selectedRowKeys.length > 0 && (
          <Popconfirm
            title={t('确认删除')}
            content={t('确定要删除选中的 {{count}} 条日志吗？', { count: selectedRowKeys.length })}
            onConfirm={() => deleteLogs(selectedRowKeys)}
          >
            <Button
              icon={<IconDeleteStroked />}
              type='danger'
            >
              {t('删除选中')} ({selectedRowKeys.length})
            </Button>
          </Popconfirm>
        )}
        {isAdminUser && (
          <Popconfirm
            title={t('确认清空')}
            content={t('确定要清空所有错误日志吗？此操作不可恢复！')}
            onConfirm={clearAllErrorLogs}
          >
            <Button
              icon={<IconDelete />}
              type='danger'
              theme='borderless'
            >
              {t('清空全部')}
            </Button>
          </Popconfirm>
        )}
        <Button
          icon={<IconRefresh />}
          loading={loading}
          onClick={refresh}
          theme='solid'
        >
          {t('刷新')}
        </Button>
      </Space>
    </div>
  );
};

export default ErrorLogsActions;
