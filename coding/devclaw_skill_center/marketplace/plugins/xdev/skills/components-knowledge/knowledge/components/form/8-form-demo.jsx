import { Form, FormInputNumber, Button } from '@coze-arch/coze-design';

const Demo = () => {
  return (
    <Form>
      <FormInputNumber
        field="age"
        label="年龄"
        placeholder="请输入年龄"
        min={0}
        max={120}
      />

      <FormInputNumber
        field="price"
        label="价格"
        placeholder="请输入价格"
        min={0}
        precision={2}
        prefix="￥"
      />

      <FormInputNumber
        field="percentage"
        label="百分比"
        placeholder="请输入百分比"
        min={0}
        max={100}
        suffix="%"
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