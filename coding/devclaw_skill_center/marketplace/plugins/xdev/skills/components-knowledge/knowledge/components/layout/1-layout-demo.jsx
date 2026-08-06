import { Layout } from '@coze-arch/coze-design';

const Demo = () => {
  return (
    <Layout>
      <Layout.Header title="页面标题" />
      <Layout.Content>
        <div style={{ padding: 20 }}>内容区域</div>
      </Layout.Content>
      <Layout.Footer>
        <div style={{ textAlign: 'center' }}>页脚内容</div>
      </Layout.Footer>
    </Layout>
  );
};

export default Demo;