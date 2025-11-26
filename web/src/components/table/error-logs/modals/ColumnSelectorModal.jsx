/*
Copyright (C) 2025 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
*/

import React from 'react';
import { Modal, Checkbox, Space, Button, Divider } from '@douyinfe/semi-ui';

const ColumnSelectorModal = (logsData) => {
  const {
    t,
    showColumnSelector,
    setShowColumnSelector,
    visibleColumns,
    handleColumnVisibilityChange,
    handleSelectAll,
    initDefaultColumns,
    COLUMN_KEYS,
  } = logsData;

  const columnLabels = {
    [COLUMN_KEYS.TIME]: t('时间'),
    [COLUMN_KEYS.CHANNEL]: t('渠道'),
    [COLUMN_KEYS.USERNAME]: t('用户名'),
    [COLUMN_KEYS.TOKEN]: t('令牌'),
    [COLUMN_KEYS.GROUP]: t('分组'),
    [COLUMN_KEYS.MODEL]: t('模型'),
    [COLUMN_KEYS.IP]: 'IP',
    [COLUMN_KEYS.ERROR_CODE]: t('错误代码'),
    [COLUMN_KEYS.ERROR_TYPE]: t('错误类型'),
    [COLUMN_KEYS.STATUS_CODE]: t('状态码'),
    [COLUMN_KEYS.CONTENT]: t('错误内容'),
    [COLUMN_KEYS.DETAILS]: t('详情'),
  };

  const allChecked = Object.values(visibleColumns).every((v) => v);
  const someChecked =
    Object.values(visibleColumns).some((v) => v) && !allChecked;

  return (
    <Modal
      title={t('选择显示列')}
      visible={showColumnSelector}
      onCancel={() => setShowColumnSelector(false)}
      footer={
        <Space>
          <Button onClick={initDefaultColumns}>{t('重置默认')}</Button>
          <Button
            theme='solid'
            onClick={() => setShowColumnSelector(false)}
          >
            {t('确定')}
          </Button>
        </Space>
      }
      width={400}
    >
      <div className='mb-4'>
        <Checkbox
          checked={allChecked}
          indeterminate={someChecked}
          onChange={(e) => handleSelectAll(e.target.checked)}
        >
          {t('全选')}
        </Checkbox>
      </div>
      <Divider margin='12px' />
      <div className='grid grid-cols-2 gap-3'>
        {Object.entries(COLUMN_KEYS).map(([key, value]) => (
          <Checkbox
            key={value}
            checked={visibleColumns[value]}
            onChange={(e) =>
              handleColumnVisibilityChange(value, e.target.checked)
            }
          >
            {columnLabels[value] || key}
          </Checkbox>
        ))}
      </div>
    </Modal>
  );
};

export default ColumnSelectorModal;
