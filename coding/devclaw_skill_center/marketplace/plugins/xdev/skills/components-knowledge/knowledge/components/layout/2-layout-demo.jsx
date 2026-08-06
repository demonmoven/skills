import { Layout } from '@coze-arch/coze-design';

const Demo = () => {
  return (
    <Layout style={{ height: '300px' }}>
      <Layout.Header title="滚动内容演示" />
      <Layout.Content scrollY>
        <div style={{ padding: 20 }}>
          {Array(20)
            .fill(null)
            .map((_, index) => (
              <div key={index} style={{ marginBottom: 20 }}>
                这是第 {index + 1} 行内容
              </div>
            ))}
        </div>
      </Layout.Content>
    </Layout>
  );
};

export default Demo;