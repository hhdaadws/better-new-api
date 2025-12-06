/*
Copyright (C) 2025 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
*/

import React from 'react';
import { Form, Button } from '@douyinfe/semi-ui';
import { IconSearch } from '@douyinfe/semi-icons';

const ErrorLogsFilters = (logsData) => {
  const { t, formInitValues, setFormApi, refresh, loading } = logsData;

  return (
    <Form
      layout='vertical'
      getFormApi={(api) => setFormApi(api)}
      initValues={formInitValues}
      onSubmit={refresh}
      autoComplete='off'
    >
      <div className='flex flex-col gap-2'>
        <div className='grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-2'>
          <div className='col-span-1 lg:col-span-2'>
            <Form.DatePicker
              field='dateRange'
              className='w-full'
              type='dateTimeRange'
              placeholder={[t('开始时间'), t('结束时间')]}
              showClear
              pure
              size='small'
            />
          </div>
          <Form.Input
            field='username'
            prefix={<IconSearch />}
            placeholder={t('用户名')}
            showClear
            pure
            size='small'
          />
          <Form.Input
            field='token_name'
            prefix={<IconSearch />}
            placeholder={t('令牌名称')}
            showClear
            pure
            size='small'
          />
          <Form.Input
            field='model_name'
            prefix={<IconSearch />}
            placeholder={t('模型名称')}
            showClear
            pure
            size='small'
          />
          <Form.Input
            field='channel'
            prefix={<IconSearch />}
            placeholder={t('渠道ID')}
            showClear
            pure
            size='small'
          />
          <Form.Input
            field='group'
            prefix={<IconSearch />}
            placeholder={t('分组')}
            showClear
            pure
            size='small'
          />
          <Form.Input
            field='ip'
            prefix={<IconSearch />}
            placeholder={t('IP地址')}
            showClear
            pure
            size='small'
          />
          <Form.Input
            field='error_code'
            prefix={<IconSearch />}
            placeholder={t('错误码')}
            showClear
            pure
            size='small'
          />
          <Form.Select
            field='status_code'
            placeholder={t('HTTP状态码')}
            showClear
            pure
            size='small'
            optionList={[
              { value: 400, label: '400 Bad Request' },
              { value: 401, label: '401 Unauthorized' },
              { value: 403, label: '403 Forbidden' },
              { value: 404, label: '404 Not Found' },
              { value: 429, label: '429 Too Many Requests' },
              { value: 500, label: '500 Internal Server Error' },
              { value: 502, label: '502 Bad Gateway' },
              { value: 503, label: '503 Service Unavailable' },
            ]}
          />
          <Form.Select
            field='error_type'
            placeholder={t('错误类型')}
            showClear
            pure
            size='small'
            optionList={[
              { value: 'upstream_error', label: t('上游错误') },
              { value: 'content_policy_violation', label: t('内容审核不通过') },
              { value: 'rate_limit', label: t('速率限制') },
              { value: 'invalid_request', label: t('无效请求') },
              { value: 'authentication_error', label: t('认证错误') },
              { value: 'timeout', label: t('超时') },
            ]}
          />
          <Form.Input
            field='content'
            prefix={<IconSearch />}
            placeholder={t('错误内容关键词')}
            showClear
            pure
            size='small'
          />
        </div>
        <div className='flex justify-end gap-2'>
          <Button
            type='tertiary'
            htmlType='submit'
            loading={loading}
            size='small'
          >
            {t('查询')}
          </Button>
        </div>
      </div>
    </Form>
  );
};

export default ErrorLogsFilters;
