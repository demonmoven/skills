import { TimePicker } from '@coze-arch/coze-design';

const Demo = () => (
  <div className="flex flex-col gap-4">
    <div>
      <h4>显示单位（默认）</h4>
      <TimePicker format="HH:mm:ss" placeholder="请选择时间" />
    </div>
    <div>
      <h4>不显示单位</h4>
      <TimePicker showUnit={false} format="HH:mm:ss" placeholder="请选择时间" />
    </div>
    <div>
      <h4>自定义格式（HH:mm）</h4>
      <TimePicker format="HH:mm" placeholder="请选择时间" />
    </div>
  </div>
);

export default Demo;