import { Progress } from '@coze-arch/coze-design';

const Demo = () => (
  <div className="space-y-4">
    <Progress size="small" percent={70} />
    <Progress size="default" percent={70} />
    <Progress size="large" percent={70} />
    <Progress size="default" percent={70} height={2} />
  </div>
);

export default Demo;