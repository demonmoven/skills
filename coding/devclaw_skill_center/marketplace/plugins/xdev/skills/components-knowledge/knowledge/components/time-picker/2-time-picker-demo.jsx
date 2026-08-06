import { TimePicker } from '@coze-arch/coze-design';

const Demo = () => (
  <div className="flex flex-col gap-4">
    <div>
      <h4>默认尺寸</h4>
      <TimePicker placeholder="请选择时间" />
    </div>
    <div>
      <h4>小尺寸</h4>
      <TimePicker size="small" placeholder="请选择时间" />
    </div>
  </div>
);

export default Demo;