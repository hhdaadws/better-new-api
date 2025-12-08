import React, { useState, useEffect } from 'react';
import { Button, Form, Card, Typography, Spin } from '@douyinfe/semi-ui';
import { API } from '../helpers';
import { showError, showSuccess } from '../helpers/utils';

const { Title, Text } = Typography;

const SubscriptionPageSetting = () => {
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [htmlContent, setHtmlContent] = useState('');

  useEffect(() => {
    fetchSetting();
  }, []);

  const fetchSetting = async () => {
    setLoading(true);
    try {
      const res = await API.get('/api/option/SubscriptionPageHTML');
      if (res.data.success) {
        setHtmlContent(res.data.data || '');
      }
    } catch (error) {
      // 可能是选项不存在，使用空字符串
      setHtmlContent('');
    } finally {
      setLoading(false);
    }
  };

  const handleSave = async () => {
    setSaving(true);
    try {
      const res = await API.put('/api/option/', {
        key: 'SubscriptionPageHTML',
        value: htmlContent,
      });
      if (res.data.success) {
        showSuccess('保存成功');
      } else {
        showError(res.data.message);
      }
    } catch (error) {
      showError('保存失败: ' + error.message);
    } finally {
      setSaving(false);
    }
  };

  if (loading) {
    return (
      <div style={{ display: 'flex', justifyContent: 'center', padding: 50 }}>
        <Spin size="large" />
      </div>
    );
  }

  return (
    <div>
      <Card>
        <Title heading={4} style={{ marginBottom: 16 }}>订阅页面设置</Title>
        <Text type="secondary" style={{ display: 'block', marginBottom: 24 }}>
          配置用户"我的订阅"页面下方显示的可订阅套餐内容，支持 HTML 格式。
        </Text>

        <Form>
          <Form.TextArea
            field="htmlContent"
            label="订阅页面 HTML 内容"
            placeholder={`<div style="padding: 20px;">
  <h3>可订阅的套餐</h3>
  <div style="display: flex; gap: 20px; flex-wrap: wrap;">
    <div style="border: 1px solid #ddd; padding: 20px; border-radius: 8px; width: 200px;">
      <h4>基础月卡</h4>
      <p>每日 10 万 tokens</p>
      <p style="font-size: 24px; font-weight: bold;">¥29/月</p>
    </div>
    <div style="border: 1px solid #ddd; padding: 20px; border-radius: 8px; width: 200px;">
      <h4>高级月卡</h4>
      <p>每日 50 万 tokens</p>
      <p style="font-size: 24px; font-weight: bold;">¥99/月</p>
    </div>
  </div>
</div>`}
            value={htmlContent}
            onChange={(value) => setHtmlContent(value)}
            rows={15}
            style={{ fontFamily: 'monospace' }}
          />

          {htmlContent && (
            <div style={{ marginTop: 24 }}>
              <Text strong style={{ display: 'block', marginBottom: 8 }}>预览效果：</Text>
              <div
                style={{
                  border: '1px solid var(--semi-color-border)',
                  borderRadius: 8,
                  padding: 16,
                  backgroundColor: 'var(--semi-color-bg-1)'
                }}
                dangerouslySetInnerHTML={{ __html: htmlContent }}
              />
            </div>
          )}

          <div style={{ marginTop: 24 }}>
            <Button
              theme="solid"
              type="primary"
              onClick={handleSave}
              loading={saving}
            >
              保存设置
            </Button>
          </div>
        </Form>
      </Card>
    </div>
  );
};

export default SubscriptionPageSetting;
