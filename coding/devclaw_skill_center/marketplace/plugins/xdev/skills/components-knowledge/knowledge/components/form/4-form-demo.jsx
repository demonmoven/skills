import { Form, FormInput, FormSelect, Button } from '@coze-arch/coze-design';
import { useState } from 'react';

const Demo = () => {
  const [formApi, setFormApi] = useState(null);

  const handleFieldChange = (field, value) => {
    if (field === 'contactType') {
      // 根据联系方式类型，重置对应的值
      if (value === 'email') {
        formApi?.setValue('phone', '');
      } else if (value === 'phone') {
        formApi?.setValue('email', '');
      }
    }
  };

  return (
    <Form getFormApi={setFormApi} onValueChange={handleFieldChange}>
      <FormSelect
        field="contactType"
        label="联系方式"
        placeholder="请选择联系方式"
        initValue="email"
      >
        <FormSelect.Option value="email">邮箱</FormSelect.Option>
        <FormSelect.Option value="phone">电话</FormSelect.Option>
      </FormSelect>

      {formApi?.getValues().contactType === 'email' && (
        <FormInput
          field="email"
          label="邮箱"
          placeholder="请输入邮箱"
          rules={[
            { required: true, message: '请输入邮箱' },
            { type: 'email', message: '请输入有效的邮箱地址' },
          ]}
        />
      )}

      {formApi?.getValues().contactType === 'phone' && (
        <FormInput
          field="phone"
          label="电话"
          placeholder="请输入电话号码"
          rules={[
            { required: true, message: '请输入电话号码' },
            { pattern: /^1[3-9]\d{9}$/, message: '请输入有效的手机号码' },
          ]}
        />
      )}

      <div style={{ marginTop: 20 }}>
        <Button type="primary" htmlType="submit">
          提交
        </Button>
      </div>
    </Form>
  );
};

export default Demo;