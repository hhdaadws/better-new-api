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

import React, { useEffect, useState, useCallback } from 'react';
import { Card, Button, Typography, Spin } from '@douyinfe/semi-ui';
import { CalendarCheck, Gift, Clock, AlertCircle } from 'lucide-react';
import { API, showError, showSuccess } from '../../helpers';
import Turnstile from 'react-turnstile';

const { Text, Title } = Typography;

const CheckinCard = ({ t, onCheckinSuccess }) => {
  const [loading, setLoading] = useState(true);
  const [checkinLoading, setCheckinLoading] = useState(false);
  const [config, setConfig] = useState(null);
  const [status, setStatus] = useState(null);
  const [turnstileToken, setTurnstileToken] = useState('');

  // 获取签到信息
  const fetchCheckinInfo = useCallback(async () => {
    try {
      const res = await API.get('/api/user/checkin');
      if (res.data.success) {
        setConfig(res.data.data.config);
        setStatus(res.data.data.status);
      } else {
        // API 返回错误，签到功能可能未启用
        console.log('签到功能不可用:', res.data.message);
        setConfig({ enabled: false });
      }
    } catch (error) {
      // 签到功能可能未启用或 Redis 未配置
      console.log('签到功能请求失败:', error);
      setConfig({ enabled: false });
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchCheckinInfo();
  }, [fetchCheckinInfo]);

  // 执行签到
  const handleCheckin = async () => {
    if (config?.turnstile_required && !turnstileToken) {
      showError(t('请完成人机验证'));
      return;
    }

    setCheckinLoading(true);
    try {
      const url = config?.turnstile_required && turnstileToken
        ? `/api/user/checkin?turnstile=${turnstileToken}`
        : '/api/user/checkin';
      const res = await API.post(url, {});
      if (res.data.success) {
        showSuccess(res.data.data.message);
        setStatus(res.data.data.status);
        if (onCheckinSuccess) {
          onCheckinSuccess();
        }
      } else {
        showError(res.data.message);
      }
    } catch (error) {
      showError(t('签到失败'));
    } finally {
      setCheckinLoading(false);
      setTurnstileToken('');
    }
  };

  // 如果签到功能未启用或正在加载，不显示
  if (loading) {
    return (
      <Card className="mb-4">
        <div className="flex justify-center items-center py-4">
          <Spin />
        </div>
      </Card>
    );
  }

  // 签到功能未启用
  if (!config || !config.enabled) {
    return null;
  }

  const hasCheckedIn = status?.has_checked_in;
  const remainingQuota = status?.remaining_quota || 0;

  return (
    <Card className="mb-4 overflow-hidden">
      <div className="flex items-start justify-between">
        <div className="flex-1">
          <div className="flex items-center gap-2 mb-3">
            <div className="p-2 rounded-lg bg-gradient-to-br from-amber-100 to-orange-100 dark:from-amber-900/30 dark:to-orange-900/30">
              <CalendarCheck className="w-5 h-5 text-amber-600 dark:text-amber-400" />
            </div>
            <Title heading={5} className="!mb-0">{t('每日签到')}</Title>
          </div>

          {hasCheckedIn ? (
            <div className="space-y-2">
              <div className="flex items-center gap-2 text-green-600 dark:text-green-400">
                <Gift className="w-4 h-4" />
                <Text>{t('今日已签到')}</Text>
              </div>
              {remainingQuota > 0 && (
                <div className="flex items-center gap-2">
                  <Clock className="w-4 h-4 text-gray-400" />
                  <Text type="tertiary">
                    {t('剩余临时额度')}: {status?.remaining_quota_label}
                  </Text>
                </div>
              )}
              <div className="flex items-center gap-2">
                <AlertCircle className="w-4 h-4 text-gray-400" />
                <Text type="tertiary" size="small">
                  {t('临时额度仅限 "free" 分组渠道使用，当日有效')}
                </Text>
              </div>
            </div>
          ) : (
            <div className="space-y-3">
              <div className="flex items-center gap-2">
                <Gift className="w-4 h-4 text-amber-500" />
                <Text>
                  {t('签到可获得')} <Text strong className="text-amber-600">{config.quota_amount_label}</Text> {t('临时额度')}
                </Text>
              </div>
              <div className="flex items-center gap-2">
                <AlertCircle className="w-4 h-4 text-gray-400" />
                <Text type="tertiary" size="small">
                  {t('临时额度仅限 "free" 分组渠道使用，当日有效')}
                </Text>
              </div>

              {config.turnstile_required && (
                <div className="mt-3">
                  <Turnstile
                    sitekey={config.turnstile_site_key}
                    onVerify={(token) => setTurnstileToken(token)}
                  />
                </div>
              )}

              <Button
                theme="solid"
                type="warning"
                onClick={handleCheckin}
                loading={checkinLoading}
                disabled={config.turnstile_required && !turnstileToken}
                icon={<CalendarCheck className="w-4 h-4" />}
              >
                {t('立即签到')}
              </Button>
            </div>
          )}
        </div>
      </div>
    </Card>
  );
};

export default CheckinCard;
