import { Switch } from '@coze-arch/coze-design';

const Demo = () => (
  <div className="flex flex-col gap-4">
    <div>
      <h4>默认状态</h4>
      <div className="flex gap-4 mt-2">
        <Switch size="default" />
        <Switch size="small" />
        <Switch size="mini" />
      </div>
    </div>
    <div>
      <h4>选中状态</h4>
      <div className="flex gap-4 mt-2">
        <Switch size="default" defaultChecked />
        <Switch size="small" defaultChecked />
        <Switch size="mini" defaultChecked />
      </div>
    </div>
  </div>
);

export default Demo;