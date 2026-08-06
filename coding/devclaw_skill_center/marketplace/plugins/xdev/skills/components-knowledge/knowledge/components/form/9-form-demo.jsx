import { Form, FormUpload, Button } from '@coze-arch/coze-design';
import { IconCozUpload } from '@coze-arch/coze-design/icons';

const Demo = () => {
  return (
    <Form>
      <FormUpload
        field="files"
        label="文件上传"
        action="//jsonplaceholder.typicode.com/posts/"
        draggable
      >
        <div
          style={{
            display: 'flex',
            flexDirection: 'column',
            alignItems: 'center',
            justifyContent: 'center',
            height: 120,
          }}
        >
          <IconCozUpload size="large" />
          <div style={{ marginTop: 8 }}>点击或拖拽文件到此区域上传</div>
        </div>
      </FormUpload>

      <div style={{ marginTop: 20 }}>
        <Button type="primary" htmlType="submit">
          提交
        </Button>
      </div>
    </Form>
  );
};

export default Demo;