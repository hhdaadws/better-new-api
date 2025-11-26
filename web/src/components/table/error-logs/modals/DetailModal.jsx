/*
Copyright (C) 2025 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
*/

import React from 'react';
import { Modal, Descriptions, Typography, Collapse, Tag, Space } from '@douyinfe/semi-ui';

const { Text, Paragraph } = Typography;

const DetailModal = (logsData) => {
  const { t, showDetailModal, setShowDetailModal, detailData } = logsData;

  if (!detailData) return null;

  const other = detailData.otherParsed || {};

  // Basic info
  const basicData = [
    { key: t('时间'), value: detailData.timestamp2string },
    { key: t('用户名'), value: detailData.username || '-' },
    { key: t('令牌'), value: detailData.token_name || '-' },
    { key: t('分组'), value: detailData.group || '-' },
    { key: t('模型'), value: detailData.model_name || '-' },
    { key: 'IP', value: detailData.ip || '-' },
    { key: t('渠道'), value: `${detailData.channel || '-'} - ${detailData.channel_name || t('未知')}` },
  ];

  // Error info
  const errorData = [
    { key: t('错误类型'), value: other.error_type || '-' },
    { key: t('错误代码'), value: other.error_code || '-' },
    { key: t('状态码'), value: other.status_code || '-' },
    { key: t('请求路径'), value: other.request_path || '-' },
    { key: t('请求方法'), value: other.request_method || '-' },
  ];

  // Admin info
  const adminData = [];
  if (other.admin_info) {
    if (other.admin_info.use_channel) {
      adminData.push({
        key: t('使用渠道'),
        value: other.admin_info.use_channel.join(' -> '),
      });
    }
    if (other.admin_info.is_multi_key) {
      adminData.push({
        key: t('多Key索引'),
        value: other.admin_info.multi_key_index,
      });
    }
  }

  return (
    <Modal
      title={t('错误日志详情')}
      visible={showDetailModal}
      onCancel={() => setShowDetailModal(false)}
      footer={null}
      width={800}
      style={{ maxHeight: '80vh' }}
      bodyStyle={{ maxHeight: '70vh', overflow: 'auto' }}
    >
      <Collapse defaultActiveKey={['basic', 'error', 'content']}>
        <Collapse.Panel header={t('基本信息')} itemKey='basic'>
          <Descriptions data={basicData} row />
        </Collapse.Panel>

        <Collapse.Panel header={t('错误信息')} itemKey='error'>
          <Descriptions data={errorData} row />
        </Collapse.Panel>

        <Collapse.Panel header={t('错误内容')} itemKey='content'>
          <Paragraph
            copyable
            style={{
              backgroundColor: 'var(--semi-color-fill-0)',
              padding: 12,
              borderRadius: 8,
              whiteSpace: 'pre-wrap',
              wordBreak: 'break-all',
              maxHeight: 200,
              overflow: 'auto',
            }}
          >
            {detailData.content || '-'}
          </Paragraph>
        </Collapse.Panel>

        {other.request_headers && (
          <Collapse.Panel header={t('请求头')} itemKey='headers'>
            <Paragraph
              copyable
              style={{
                backgroundColor: 'var(--semi-color-fill-0)',
                padding: 12,
                borderRadius: 8,
                whiteSpace: 'pre-wrap',
                wordBreak: 'break-all',
                maxHeight: 300,
                overflow: 'auto',
                fontFamily: 'monospace',
                fontSize: 12,
              }}
            >
              {JSON.stringify(other.request_headers, null, 2)}
            </Paragraph>
          </Collapse.Panel>
        )}

        {other.request_body && (
          <Collapse.Panel header={t('请求体')} itemKey='body'>
            <Paragraph
              copyable
              style={{
                backgroundColor: 'var(--semi-color-fill-0)',
                padding: 12,
                borderRadius: 8,
                whiteSpace: 'pre-wrap',
                wordBreak: 'break-all',
                maxHeight: 400,
                overflow: 'auto',
                fontFamily: 'monospace',
                fontSize: 12,
              }}
            >
              {other.request_body}
            </Paragraph>
          </Collapse.Panel>
        )}

        {adminData.length > 0 && (
          <Collapse.Panel header={t('管理员信息')} itemKey='admin'>
            <Descriptions data={adminData} row />
          </Collapse.Panel>
        )}
      </Collapse>
    </Modal>
  );
};

export default DetailModal;
