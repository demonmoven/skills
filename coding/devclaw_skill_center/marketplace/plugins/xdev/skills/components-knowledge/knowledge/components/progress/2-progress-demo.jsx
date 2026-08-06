import { Progress } from '@coze-arch/coze-design';

const Demo = () => (
  <div className="space-y-4">
    <Progress type="line" percent={70} />
    <Progress type="line" percent={70} height={4} />
    <Progress type="line" percent={70} stroke="var(--coz-fg-hglt-green)" />
    <Progress
      type="line"
      percent={70}
      stroke="var(--coz-fg-hglt-green)"
      style={{ height: 10 }}
    />
  </div>
);

export default Demo;