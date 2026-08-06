import { Form, FormInput, Button } from '@coze-arch/coze-design';

const Demo = () => {
  return (
    <div style={{ display: 'flex', gap: 20 }}>
      <div style={{ flex: 1 }}>
        <h4>垂直布局（默认）</h4>
        <Form>
          <FormInput field="name" label="姓名" placeholder="请输入姓名" />
          <FormInput field="email" label="邮箱" placeholder="请输入邮箱" />
          <Button type="primary" htmlType="submit">
            提交
          </Button>
        </Form>
      </div>
      <div style={{ flex: 1 }}>
        <h4>水平布局</h4>
        <Form layout="horizontal">
          <FormInput field="name" label="姓名" placeholder="请输入姓名" />
          <FormInput field="email" label="邮箱" placeholder="请输入邮箱" />
          <Button type="primary" htmlType="submit">
            提交
          </Button>
        </Form>
      </div>
    </div>
  );
};

export default Demo;