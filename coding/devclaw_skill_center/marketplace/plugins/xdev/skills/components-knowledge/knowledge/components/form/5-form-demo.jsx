import { Form, FormInput, Button } from '@coze-arch/coze-design';

const Demo = () => {
  return (
    <Form>
      <FormInput field="username" label="用户名" placeholder="请输入用户名" />
      <FormInput
        field="password"
        label="密码"
        type="password"
        placeholder="请输入密码"
      />
      <FormInput
        field="disabled"
        label="禁用状态"
        disabled
        placeholder="禁用状态"
        initValue="这是禁用状态的输入框"
      />
      <FormInput
        field="prefix"
        label="带前缀的输入框"
        prefix="￥"
        placeholder="请输入金额"
      />
      <FormInput
        field="suffix"
        label="带后缀的输入框"
        suffix=".com"
        placeholder="请输入域名"
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