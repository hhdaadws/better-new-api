/*
Copyright (C) 2025 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
*/

import React from 'react';
import { Tag, Typography, Tooltip, Popconfirm, Button } from '@douyinfe/semi-ui';
import { IconDelete } from '@douyinfe/semi-icons';

const { Text } = Typography;

export const getErrorLogsColumns = ({
  t,
  COLUMN_KEYS,
  copyText,
  showDetailFunc,
  deleteLog,
  isAdminUser,
}) => {
  return [
    {
      title: t('时间'),
      dataIndex: 'timestamp2string',
      key: COLUMN_KEYS.TIME,
      width: 170,
      fixed: 'left',
      render: (text) => <Text copyable>{text}</Text>,
    },
    {
      title: t('渠道'),
      dataIndex: 'channel',
      key: COLUMN_KEYS.CHANNEL,
      width: 100,
      render: (text, record) => (
        <Tooltip content={record.channel_name || t('未知')}>
          <Tag color="blue">{text}</Tag>
        </Tooltip>
      ),
    },
    {
      title: t('用户名'),
      dataIndex: 'username',
      key: COLUMN_KEYS.USERNAME,
      width: 120,
      render: (text) => (
        <Text
          ellipsis={{ showTooltip: true }}
          style={{ maxWidth: 100 }}
          copyable
        >
          {text || '-'}
        </Text>
      ),
    },
    {
      title: t('令牌'),
      dataIndex: 'token_name',
      key: COLUMN_KEYS.TOKEN,
      width: 120,
      render: (text) => (
        <Text ellipsis={{ showTooltip: true }} style={{ maxWidth: 100 }}>
          {text || '-'}
        </Text>
      ),
    },
    {
      title: t('分组'),
      dataIndex: 'group',
      key: COLUMN_KEYS.GROUP,
      width: 100,
      render: (text) => <Tag>{text || '-'}</Tag>,
    },
    {
      title: t('模型'),
      dataIndex: 'model_name',
      key: COLUMN_KEYS.MODEL,
      width: 180,
      render: (text) => (
        <Text
          ellipsis={{ showTooltip: true }}
          style={{ maxWidth: 160 }}
          copyable
        >
          {text || '-'}
        </Text>
      ),
    },
    {
      title: 'IP',
      dataIndex: 'ip',
      key: COLUMN_KEYS.IP,
      width: 140,
      render: (text) => (
        <Text
          copyable
          style={{ cursor: 'pointer' }}
          onClick={(e) => text && copyText(e, text)}
        >
          {text || '-'}
        </Text>
      ),
    },
    {
      title: t('错误代码'),
      dataIndex: 'otherParsed',
      key: COLUMN_KEYS.ERROR_CODE,
      width: 120,
      render: (other) => (
        <Tag color="red">{other?.error_code || '-'}</Tag>
      ),
    },
    {
      title: t('错误类型'),
      dataIndex: 'otherParsed',
      key: COLUMN_KEYS.ERROR_TYPE,
      width: 150,
      render: (other) => (
        <Text ellipsis={{ showTooltip: true }} style={{ maxWidth: 130 }}>
          {other?.error_type || '-'}
        </Text>
      ),
    },
    {
      title: t('状态码'),
      dataIndex: 'otherParsed',
      key: COLUMN_KEYS.STATUS_CODE,
      width: 90,
      render: (other) => {
        const statusCode = other?.status_code;
        let color = 'grey';
        if (statusCode >= 400 && statusCode < 500) color = 'orange';
        if (statusCode >= 500) color = 'red';
        return <Tag color={color}>{statusCode || '-'}</Tag>;
      },
    },
    {
      title: t('错误内容'),
      dataIndex: 'content',
      key: COLUMN_KEYS.CONTENT,
      width: 250,
      render: (text) => (
        <Text
          ellipsis={{ showTooltip: true }}
          style={{ maxWidth: 230, color: '#f5222d' }}
        >
          {text || '-'}
        </Text>
      ),
    },
    {
      title: t('操作'),
      dataIndex: 'id',
      key: COLUMN_KEYS.DETAILS,
      width: isAdminUser ? 120 : 80,
      fixed: 'right',
      render: (id, record) => (
        <div className='flex items-center gap-2'>
          <Text
            link
            onClick={() => showDetailFunc(record)}
            style={{ cursor: 'pointer' }}
          >
            {t('查看')}
          </Text>
          {isAdminUser && deleteLog && (
            <Popconfirm
              title={t('确认删除')}
              content={t('确定要删除这条日志吗？')}
              onConfirm={() => deleteLog(id)}
            >
              <Button
                icon={<IconDelete />}
                type='danger'
                theme='borderless'
                size='small'
              />
            </Popconfirm>
          )}
        </div>
      ),
    },
  ];
};
