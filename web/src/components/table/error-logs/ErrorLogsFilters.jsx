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
