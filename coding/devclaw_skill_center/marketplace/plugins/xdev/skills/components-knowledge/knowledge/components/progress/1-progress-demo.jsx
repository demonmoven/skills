import { Progress } from '@coze-arch/coze-design';

const Demo = () => (
  <div className="space-y-4">
    <Progress percent={30} />
    <Progress percent={50} />
    <Progress percent={70} />
  </div>
);

export default Demo;