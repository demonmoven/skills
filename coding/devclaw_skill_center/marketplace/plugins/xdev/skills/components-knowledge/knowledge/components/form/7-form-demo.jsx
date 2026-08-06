import { Form, FormTextArea, Button } from '@coze-arch/coze-design';

const Demo = () => {
  return (
    <Form>
      <FormTextArea
        field="description"
        label="描述"
        placeholder="请输入描述信息"
      />

      <FormTextArea
        field="feedback"
        label="反馈"
        placeholder="请输入反馈信息"
        rows={6}
        showClear
      />

      <FormTextArea
        field="limited"
        label="限制字数"
        placeholder="请输入内容，最多100个字符"
        maxCount={100}
        showClear
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