import { TimePicker } from '@coze-arch/coze-design';

const Demo = () => (
  <div className="flex flex-col gap-4">
    <div>
      <h4>显示图标（默认）</h4>
      <TimePicker placeholder="请选择时间" />
    </div>
    <div>
      <h4>不显示图标</h4>
      <TimePicker showIcon={false} placeholder="请选择时间" />
    </div>
  </div>
);

export default Demo;