/*
Copyright (C) 2025 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
*/

import React from 'react';
import { Button, Typography, Space, Switch } from '@douyinfe/semi-ui';
import { IconSetting, IconRefresh } from '@douyinfe/semi-icons';

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
