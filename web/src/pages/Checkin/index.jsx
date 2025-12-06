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

import React, { useEffect, useState } from 'react';
import {
  Card,
  Form,
  Button,
  Typography,
  InputNumber,
  Switch,
  Banner,
  Descriptions,
  Spin,
} from '@douyinfe/semi-ui';
import { useTranslation } from 'react-i18next';
import { API, showError, showSuccess } from '../../helpers';

const { Title, Text } = Typography;

const CheckinSetting = () => {
  const { t } = useTranslation();
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [config, setConfig] = useState({
    enabled: true,
    quota_amount: 500000,
    group: 'free',
  });

  // Fetch current config
  const fetchConfig = async () => {
    setLoading(true);
    try {
      const res = await API.get('/api/option/checkin');
      if (res.data.success) {
        setConfig(res.data.data);
      } else {
        showError(res.data.message || t('获取签到配置失败'));
      }
    } catch (error) {
      showError(t('获取签到配置失败'));
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchConfig();
  }, []);

  // Save config
  const handleSave = async () => {
    setSaving(true);
    try {
      const res = await API.put('/api/option/checkin', config);
      if (res.data.success) {
        showSuccess(t('签到配置保存成功'));
      } else {
        showError(res.data.message || t('保存签到配置失败'));
      }
    } catch (error) {
      showError(t('保存签到配置失败'));
    } finally {
      setSaving(false);
    }
  };

  // Format quota for display
  const formatQuota = (quota) => {
    // Assuming QuotaPerUnit = 500000 for $1
    const quotaPerUnit = 500000;
    const dollars = quota / quotaPerUnit;
    return `$${dollars.toFixed(2)}`;
  };

  if (loading) {
    return (
      <div className='mt-[60px] px-4 flex justify-center items-center min-h-[400px]'>
        <Spin size='large' />
      </div>
    );
  }

  return (
    <div className='mt-[60px] px-4'>
      <Title heading={3} className='mb-4'>
        {t('签到管理')}
      </Title>

      <Banner
        type='info'
        description={
          <div>
            <p>{t('每日签到功能允许用户每天签到一次获得临时免费额度。')}</p>
            <p>{t('签到获得的额度仅限于 "free" 分组的渠道使用，当日有效（新加坡时间 UTC+8）。')}</p>
            <p><strong>{t('注意：管理员需要创建名为 "free" 的渠道分组，并将希望提供给签到用户的渠道分配到该分组。')}</strong></p>
          </div>
        }
        className='mb-4'
      />

      <Card className='mb-4'>
        <Title heading={5} className='mb-4'>
          {t('签到配置')}
        </Title>

        <Form
          labelPosition='left'
          labelWidth={180}
          className='max-w-xl'
        >
          <Form.Slot label={t('启用签到功能')}>
            <Switch
              checked={config.enabled}
              onChange={(checked) => setConfig({ ...config, enabled: checked })}
            />
          </Form.Slot>

          <Form.Slot label={t('签到额度')}>
            <div className='flex items-center gap-4'>
              <InputNumber
                value={config.quota_amount}
                onChange={(value) => setConfig({ ...config, quota_amount: value || 0 })}
                min={0}
                step={100000}
                style={{ width: 200 }}
                disabled={!config.enabled}
              />
              <Text type='tertiary'>
                {t('约等于')} {formatQuota(config.quota_amount)}
              </Text>
            </div>
          </Form.Slot>

          <Form.Slot label={t('可用分组')}>
            <Text strong>{config.group}</Text>
            <Text type='tertiary' className='ml-2'>
              ({t('固定为 "free" 分组，不可修改')})
            </Text>
          </Form.Slot>

          <Form.Slot label={t('Turnstile 验证')}>
            <Text type='tertiary'>
              {t('签到时会自动使用系统配置的 Turnstile 验证（如已启用）')}
            </Text>
          </Form.Slot>
        </Form>

        <div className='mt-6'>
          <Button
            type='primary'
            onClick={handleSave}
            loading={saving}
            disabled={!config.enabled && config.quota_amount === 0}
          >
            {t('保存配置')}
          </Button>
        </div>
      </Card>

      <Card>
        <Title heading={5} className='mb-4'>
          {t('预设额度参考')}
        </Title>

        <Descriptions
          data={[
            { key: '$0.10', value: '50,000' },
            { key: '$0.50', value: '250,000' },
            { key: '$1.00', value: '500,000' },
            { key: '$2.00', value: '1,000,000' },
            { key: '$5.00', value: '2,500,000' },
            { key: '$10.00', value: '5,000,000' },
          ]}
          row
          className='mb-4'
        />

        <Text type='tertiary'>
          {t('以上为额度与美元的换算参考（基于 QuotaPerUnit = 500,000）')}
        </Text>
      </Card>
    </div>
  );
};

export default CheckinSetting;
