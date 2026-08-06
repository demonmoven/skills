import { Breadcrumb } from '@coze-arch/coze-design';

const Demo = () => (
  <>
    <Breadcrumb style={{ marginBottom: 20 }}>
      <Breadcrumb.Item>Coze Design</Breadcrumb.Item>
      <Breadcrumb.Item>Components</Breadcrumb.Item>
      <Breadcrumb.Item>Default Size</Breadcrumb.Item>
    </Breadcrumb>

    <Breadcrumb size="small">
      <Breadcrumb.Item>Coze Design</Breadcrumb.Item>
      <Breadcrumb.Item>Components</Breadcrumb.Item>
      <Breadcrumb.Item>Small Size</Breadcrumb.Item>
    </Breadcrumb>
  </>
);

export default Demo;