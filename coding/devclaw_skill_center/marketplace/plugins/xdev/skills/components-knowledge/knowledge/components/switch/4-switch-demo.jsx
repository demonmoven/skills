import { Switch } from '@coze-arch/coze-design';

const Demo = () => (
  <div className="flex flex-col gap-4">
    <div>
      <h4>禁用状态</h4>
      <div className="flex gap-4 mt-2">
        <Switch disabled />
        <Switch checked disabled />
      </div>
    </div>
    <div>
      <h4>加载状态</h4>
      <div className="flex gap-4 mt-2">
        <Switch loading />
        <Switch checked loading />
      </div>
    </div>
    <div>
      <h4>禁用且加载</h4>
      <div className="flex gap-4 mt-2">
        <Switch disabled loading />
        <Switch checked disabled loading />
      </div>
    </div>
  </div>
);

export default Demo;