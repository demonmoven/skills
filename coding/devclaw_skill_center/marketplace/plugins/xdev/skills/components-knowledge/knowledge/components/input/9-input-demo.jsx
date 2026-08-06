import { Form } from '@coze-arch/coze-design';
import { FormInput } from '@coze-arch/coze-design';
import { Button } from '@coze-arch/coze-design';

const Demo = () => {
  const handleSubmit = values => {
    console.log('表单值：', values);
  };

  return (
    <Form onSubmit={handleSubmit} style={{ width: 400 }}>
      <FormInput
        field="username"
        label="用户名"
        rules={[{ required: true, message: '请输入用户名' }]}
      />
      <FormInput
        field="email"
        label="邮箱"
        rules={[
          { required: true, message: '请输入邮箱' },
          { type: 'email', message: '请输入有效的邮箱地址' },
        ]}
      />
      <FormInput
        field="password"
        label="密码"
        type="password"
        size="small"
        rules={[{ required: true, message: '请输入密码' }]}
      />
      <div style={{ marginTop: 16 }}>
        <Button type="submit" color="brand">
          提交
        </Button>
      </div>
    </Form>
  );
};

export default Demo;