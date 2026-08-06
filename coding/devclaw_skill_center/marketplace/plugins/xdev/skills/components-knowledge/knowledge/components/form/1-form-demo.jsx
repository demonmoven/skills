import {
  Form,
  FormInput,
  FormSelect,
  FormTextArea,
  Button,
} from '@coze-arch/coze-design';
import { useState } from 'react';

const Demo = () => {
  const [formApi, setFormApi] = useState(null);

  const handleSubmit = values => {
    console.log('表单提交的值：', values);
  };

  return (
    <Form onSubmit={handleSubmit} getFormApi={setFormApi}>
      <FormInput
        field="name"
        label="姓名"
        placeholder="请输入姓名"
        rules={[{ required: true, message: '请输入姓名' }]}
      />
      <FormSelect
        field="gender"
        label="性别"
        placeholder="请选择性别"
        rules={[{ required: true, message: '请选择性别' }]}
      >
        <FormSelect.Option value="male">男</FormSelect.Option>
        <FormSelect.Option value="female">女</FormSelect.Option>
      </FormSelect>
      <FormTextArea
        field="description"
        label="个人简介"
        placeholder="请输入个人简介"
      />
      <div style={{ marginTop: 20 }}>
        <Button type="primary" htmlType="submit">
          提交
        </Button>
        <Button style={{ marginLeft: 8 }} onClick={() => formApi?.reset()}>
          重置
        </Button>
      </div>
    </Form>
  );
};

export default Demo;