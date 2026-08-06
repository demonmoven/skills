import { Progress } from '@coze-arch/coze-design';

const Demo = () => (
  <div className="flex items-center space-x-8 gap-4">
    <Progress type="circle" percent={70} />
    <Progress type="circle" percent={70} width={50} strokeWidth={4} />
    <Progress type="circle" percent={70} stroke="var(--coz-fg-hglt-green)" />
  </div>
);

export default Demo;