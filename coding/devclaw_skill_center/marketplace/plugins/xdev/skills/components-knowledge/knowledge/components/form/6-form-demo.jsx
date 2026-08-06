import { Form, FormSelect, Button } from '@coze-arch/coze-design';

const Demo = () => {
  return (
    <Form>
      <FormSelect field="single" label="单选" placeholder="请选择一个选项">
        <FormSelect.Option value="option1">选项一</FormSelect.Option>
        <FormSelect.Option value="option2">选项二</FormSelect.Option>
        <FormSelect.Option value="option3">选项三</FormSelect.Option>
      </FormSelect>

      <FormSelect
        field="multiple"
        label="多选"
        placeholder="请选择多个选项"
        multiple
      >
        <FormSelect.Option value="option1">选项一</FormSelect.Option>
        <FormSelect.Option value="option2">选项二</FormSelect.Option>
        <FormSelect.Option value="option3">选项三</FormSelect.Option>
      </FormSelect>

      <FormSelect field="group" label="分组选择" placeholder="请选择一个选项">
        <FormSelect.OptGroup label="分组一">
          <FormSelect.Option value="group1-option1">
            分组一选项一
          </FormSelect.Option>
          <FormSelect.Option value="group1-option2">
            分组一选项二
          </FormSelect.Option>
        </FormSelect.OptGroup>
        <FormSelect.OptGroup label="分组二">
          <FormSelect.Option value="group2-option1">
            分组二选项一
          </FormSelect.Option>
          <FormSelect.Option value="group2-option2">
            分组二选项二
          </FormSelect.Option>
        </FormSelect.OptGroup>
      </FormSelect>

      <div style={{ marginTop: 20 }}>
        <Button type="primary" htmlType="submit">
          提交
        </Button>
      </div>
    </Form>
  );
};

export default Demo;