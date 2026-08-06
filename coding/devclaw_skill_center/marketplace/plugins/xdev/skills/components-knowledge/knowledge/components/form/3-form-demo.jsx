import { Form, FormInput, Button } from '@coze-arch/coze-design';

const Demo = () => {
  const handleSubmit = values => {
    console.log('表单提交的值：', values);
  };

  return (
    <Form onSubmit={handleSubmit}>
      <FormInput
        field="username"
        label="用户名"
        placeholder="请输入用户名"
        rules={[
          { required: true, message: '请输入用户名' },
          { min: 3, message: '用户名至少3个字符' },
          { max: 20, message: '用户名最多20个字符' },
        ]}
      />
      <FormInput
        field="email"
        label="邮箱"
        placeholder="请输入邮箱"
        rules={[
          { required: true, message: '请输入邮箱' },
          { type: 'email', message: '请输入有效的邮箱地址' },
        ]}
      />
      <FormInput
        field="password"
        label="密码"
        type="password"
        placeholder="请输入密码"
        rules={[
          { required: true, message: '请输入密码' },
          {
            validator: (rule, value) => value && value.length >= 6,
            message: '密码长度至少为6位',
          },
        ]}
      />
      <div style={{ marginTop: 20 }}>
        <Button type="primary" htmlType="submit">
          提交
        </Button>
      </div>
    </Form>
  );
};

export default Demo;