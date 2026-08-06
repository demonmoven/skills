import { Badge } from '@coze-arch/coze-design';

const Demo = () => (
  <div className="flex flex-row gap-4 p-4">
    <Badge />
    <Badge count="beta" type="alt" />
    <Badge
      type="mini"
      countStyle={{ backgroundColor: 'var(--coz-fg-hglt-green)' }}
    />
  </div>
);

export default Demo;