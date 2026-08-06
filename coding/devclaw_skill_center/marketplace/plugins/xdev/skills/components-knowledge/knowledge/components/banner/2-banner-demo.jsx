import { Banner } from '@coze-arch/coze-design';

const Demo = () => (
  <div className="space-y-4">
    <Banner type="info" description="这是一条信息提示" />
    <Banner type="success" description="这是一条成功提示" />
    <Banner type="warning" description="这是一条警告提示" />
    <Banner type="danger" description="这是一条危险提示" />
  </div>
);

export default Demo;
