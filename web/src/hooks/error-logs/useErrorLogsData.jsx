/*
Copyright (C) 2025 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
*/

import { useState, useEffect } from 'react';
import { useTranslation } from 'react-i18next';
import { Modal } from '@douyinfe/semi-ui';
import {
  API,
  getTodayStartTimestamp,
  isAdmin,
  showError,
  showSuccess,
  timestamp2string,
  getLogOther,
  copy,
} from '../../helpers';
import { ITEMS_PER_PAGE } from '../../constants';
import { useTableCompactMode } from '../common/useTableCompactMode';

// Error log type constant (matches backend LogTypeError = 5)
const LOG_TYPE_ERROR = 5;

export const useErrorLogsData = () => {
  const { t } = useTranslation();

  // Define column keys for selection
  const COLUMN_KEYS = {
    TIME: 'time',
    CHANNEL: 'channel',
    USERNAME: 'username',
    TOKEN: 'token',
    GROUP: 'group',
    MODEL: 'model',
    IP: 'ip',
    ERROR_CODE: 'error_code',
    ERROR_TYPE: 'error_type',
    STATUS_CODE: 'status_code',
    CONTENT: 'content',
    DETAILS: 'details',
  };

  // Basic state
  const [logs, setLogs] = useState([]);
  const [expandData, setExpandData] = useState({});
  const [loading, setLoading] = useState(false);
  const [activePage, setActivePage] = useState(1);
  const [logCount, setLogCount] = useState(0);
  const [pageSize, setPageSize] = useState(ITEMS_PER_PAGE);

  // User and admin
  const isAdminUser = isAdmin();
  const STORAGE_KEY = 'error-logs-table-columns';

  // Form state
  const [formApi, setFormApi] = useState(null);
  const [selectedRowKeys, setSelectedRowKeys] = useState([]);
  let now = new Date();
  const formInitValues = {
    username: '',
    token_name: '',
    model_name: '',
    channel: '',
    group: '',
    ip: '',
    error_code: '',
    status_code: '',
    error_type: '',
    content: '',
    dateRange: [
      timestamp2string(getTodayStartTimestamp()),
      timestamp2string(now.getTime() / 1000 + 3600),
    ],
  };

  // Column visibility state
  const [visibleColumns, setVisibleColumns] = useState({});
  const [showColumnSelector, setShowColumnSelector] = useState(false);

  // Compact mode
  const [compactMode, setCompactMode] = useTableCompactMode('error-logs');

  // Detail modal state
  const [showDetailModal, setShowDetailModal] = useState(false);
  const [detailData, setDetailData] = useState(null);

  // Load saved column preferences from localStorage
  useEffect(() => {
    const savedColumns = localStorage.getItem(STORAGE_KEY);
    if (savedColumns) {
      try {
        const parsed = JSON.parse(savedColumns);
        const defaults = getDefaultColumnVisibility();
        const merged = { ...defaults, ...parsed };
        setVisibleColumns(merged);
      } catch (e) {
        console.error('Failed to parse saved column preferences', e);
        initDefaultColumns();
      }
    } else {
      initDefaultColumns();
    }
  }, []);

  // Get default column visibility
  const getDefaultColumnVisibility = () => {
    return {
      [COLUMN_KEYS.TIME]: true,
      [COLUMN_KEYS.CHANNEL]: true,
      [COLUMN_KEYS.USERNAME]: true,
      [COLUMN_KEYS.TOKEN]: true,
      [COLUMN_KEYS.GROUP]: true,
      [COLUMN_KEYS.MODEL]: true,
      [COLUMN_KEYS.IP]: true,
      [COLUMN_KEYS.ERROR_CODE]: true,
      [COLUMN_KEYS.ERROR_TYPE]: true,
      [COLUMN_KEYS.STATUS_CODE]: true,
      [COLUMN_KEYS.CONTENT]: true,
      [COLUMN_KEYS.DETAILS]: true,
    };
  };

  // Initialize default column visibility
  const initDefaultColumns = () => {
    const defaults = getDefaultColumnVisibility();
    setVisibleColumns(defaults);
    localStorage.setItem(STORAGE_KEY, JSON.stringify(defaults));
  };

  // Handle column visibility change
  const handleColumnVisibilityChange = (columnKey, checked) => {
    const updatedColumns = { ...visibleColumns, [columnKey]: checked };
    setVisibleColumns(updatedColumns);
  };

  // Handle "Select All" checkbox
  const handleSelectAll = (checked) => {
    const allKeys = Object.keys(COLUMN_KEYS).map((key) => COLUMN_KEYS[key]);
    const updatedColumns = {};
    allKeys.forEach((key) => {
      updatedColumns[key] = checked;
    });
    setVisibleColumns(updatedColumns);
  };

  // Persist column settings
  useEffect(() => {
    if (Object.keys(visibleColumns).length > 0) {
      localStorage.setItem(STORAGE_KEY, JSON.stringify(visibleColumns));
    }
  }, [visibleColumns]);

  // Get form values helper
  const getFormValues = () => {
    const formValues = formApi ? formApi.getValues() : {};

    let start_timestamp = timestamp2string(getTodayStartTimestamp());
    let end_timestamp = timestamp2string(now.getTime() / 1000 + 3600);

    if (
      formValues.dateRange &&
      Array.isArray(formValues.dateRange) &&
      formValues.dateRange.length === 2
    ) {
      start_timestamp = formValues.dateRange[0];
      end_timestamp = formValues.dateRange[1];
    }

    return {
      username: formValues.username || '',
      token_name: formValues.token_name || '',
      model_name: formValues.model_name || '',
      start_timestamp,
      end_timestamp,
      channel: formValues.channel || '',
      group: formValues.group || '',
      ip: formValues.ip || '',
      error_code: formValues.error_code || '',
      status_code: formValues.status_code || '',
      error_type: formValues.error_type || '',
      content: formValues.content || '',
    };
  };

  // Show detail modal
  const showDetailFunc = (record) => {
    setDetailData(record);
    setShowDetailModal(true);
  };

  // Format logs data
  const setLogsFormat = (logs) => {
    let expandDatesLocal = {};
    for (let i = 0; i < logs.length; i++) {
      logs[i].timestamp2string = timestamp2string(logs[i].created_at);
      logs[i].key = logs[i].id;
      let other = getLogOther(logs[i].other);
      logs[i].otherParsed = other;
      let expandDataLocal = [];

      // Channel info
      if (logs[i].channel) {
        expandDataLocal.push({
          key: t('渠道信息'),
          value: `${logs[i].channel} - ${logs[i].channel_name || '[未知]'}`,
        });
      }

      // Error details
      if (other?.error_type) {
        expandDataLocal.push({
          key: t('错误类型'),
          value: other.error_type,
        });
      }
      if (other?.error_code) {
        expandDataLocal.push({
          key: t('错误代码'),
          value: other.error_code,
        });
      }
      if (other?.status_code) {
        expandDataLocal.push({
          key: t('状态码'),
          value: other.status_code,
        });
      }
      if (other?.request_path) {
        expandDataLocal.push({
          key: t('请求路径'),
          value: other.request_path,
        });
      }
      if (other?.request_method) {
        expandDataLocal.push({
          key: t('请求方法'),
          value: other.request_method,
        });
      }

      // Request headers
      if (other?.request_headers) {
        expandDataLocal.push({
          key: t('请求头'),
          value: JSON.stringify(other.request_headers, null, 2),
        });
      }

      // Request body (truncated display)
      if (other?.request_body) {
        let bodyPreview = other.request_body;
        if (bodyPreview.length > 200) {
          bodyPreview = bodyPreview.substring(0, 200) + '...';
        }
        expandDataLocal.push({
          key: t('请求体'),
          value: bodyPreview,
        });
      }

      // Admin info
      if (other?.admin_info) {
        if (other.admin_info.use_channel) {
          expandDataLocal.push({
            key: t('使用渠道'),
            value: other.admin_info.use_channel.join(' -> '),
          });
        }
        if (other.admin_info.is_multi_key) {
          expandDataLocal.push({
            key: t('多Key'),
            value: `Key #${other.admin_info.multi_key_index}`,
          });
        }
      }

      expandDatesLocal[logs[i].key] = expandDataLocal;
    }

    setExpandData(expandDatesLocal);
    setLogs(logs);
  };

  // Load logs function
  const loadLogs = async (startIdx, pageSizeParam) => {
    setLoading(true);

    const {
      username,
      token_name,
      model_name,
      start_timestamp,
      end_timestamp,
      channel,
      group,
      ip,
      error_code,
      status_code,
      error_type,
      content,
    } = getFormValues();

    let localStartTimestamp = Date.parse(start_timestamp) / 1000;
    let localEndTimestamp = Date.parse(end_timestamp) / 1000;

    let url = `/api/log/?p=${startIdx}&page_size=${pageSizeParam}&type=${LOG_TYPE_ERROR}&username=${username}&token_name=${token_name}&model_name=${model_name}&start_timestamp=${localStartTimestamp}&end_timestamp=${localEndTimestamp}&channel=${channel}&group=${group}&ip=${ip}&error_code=${error_code}&status_code=${status_code}&error_type=${error_type}&content=${content}`;
    url = encodeURI(url);

    try {
      const res = await API.get(url);
      const { success, message, data } = res.data;
      if (success) {
        const newPageData = data.items;
        setActivePage(data.page);
        setPageSize(data.page_size);
        setLogCount(data.total);
        setLogsFormat(newPageData);
      } else {
        showError(message);
      }
    } catch (error) {
      showError(error.message || 'Failed to load error logs');
    }
    setLoading(false);
  };

  // Page handlers
  const handlePageChange = (page) => {
    setActivePage(page);
    loadLogs(page, pageSize).then((r) => {});
  };

  const handlePageSizeChange = async (size) => {
    localStorage.setItem('error-logs-page-size', size + '');
    setPageSize(size);
    setActivePage(1);
    loadLogs(1, size)
      .then()
      .catch((reason) => {
        showError(reason);
      });
  };

  // Refresh function
  const refresh = async () => {
    setActivePage(1);
    await loadLogs(1, pageSize);
  };

  // Delete single log
  const deleteLog = async (id) => {
    try {
      const res = await API.delete(`/api/log/${id}`);
      const { success, message } = res.data;
      if (success) {
        showSuccess(t('删除成功'));
        await refresh();
      } else {
        showError(message);
      }
    } catch (error) {
      showError(error.message || t('删除失败'));
    }
  };

  // Batch delete logs
  const deleteLogs = async (ids) => {
    if (!ids || ids.length === 0) {
      showError(t('请选择要删除的日志'));
      return;
    }
    try {
      const res = await API.delete('/api/log/batch', { data: { ids } });
      const { success, message, data } = res.data;
      if (success) {
        showSuccess(t('成功删除') + ` ${data} ` + t('条日志'));
        setSelectedRowKeys([]);
        await refresh();
      } else {
        showError(message);
      }
    } catch (error) {
      showError(error.message || t('批量删除失败'));
    }
  };

  // Clear all error logs
  const clearAllErrorLogs = async () => {
    try {
      const res = await API.delete('/api/log/error/clear');
      const { success, message, data } = res.data;
      if (success) {
        showSuccess(t('成功清空') + ` ${data} ` + t('条错误日志'));
        setSelectedRowKeys([]);
        await refresh();
      } else {
        showError(message);
      }
    } catch (error) {
      showError(error.message || t('清空失败'));
    }
  };

  // Copy text function
  const copyText = async (e, text) => {
    e.stopPropagation();
    if (await copy(text)) {
      showSuccess(t('已复制') + '：' + text);
    } else {
      Modal.error({ title: t('无法复制到剪贴板，请手动复制'), content: text });
    }
  };

  // Initialize data
  useEffect(() => {
    const localPageSize =
      parseInt(localStorage.getItem('error-logs-page-size')) || ITEMS_PER_PAGE;
    setPageSize(localPageSize);
    loadLogs(1, localPageSize)
      .then()
      .catch((reason) => {
        showError(reason);
      });
  }, []);

  // Check if any record has expandable content
  const hasExpandableRows = () => {
    return logs.some(
      (log) => expandData[log.key] && expandData[log.key].length > 0,
    );
  };

  return {
    // Basic state
    logs,
    expandData,
    loading,
    activePage,
    logCount,
    pageSize,
    isAdminUser,

    // Form state
    formApi,
    setFormApi,
    formInitValues,
    getFormValues,

    // Selection state
    selectedRowKeys,
    setSelectedRowKeys,

    // Column visibility
    visibleColumns,
    showColumnSelector,
    setShowColumnSelector,
    handleColumnVisibilityChange,
    handleSelectAll,
    initDefaultColumns,
    COLUMN_KEYS,

    // Compact mode
    compactMode,
    setCompactMode,

    // Detail modal
    showDetailModal,
    setShowDetailModal,
    detailData,
    showDetailFunc,

    // Functions
    loadLogs,
    handlePageChange,
    handlePageSizeChange,
    refresh,
    copyText,
    setLogsFormat,
    hasExpandableRows,

    // Delete functions
    deleteLog,
    deleteLogs,
    clearAllErrorLogs,

    // Translation
    t,
  };
};
