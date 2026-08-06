import { TimePicker } from '@coze-arch/coze-design';

const Demo = () => (
  <div className="flex flex-col gap-4">
    <div>
      <h4>单个时间</h4>
      <TimePicker placeholder="请选择时间" />
    </div>
    <div>
      <h4>时间范围</h4>
      <TimePicker type="timeRange" placeholder={['开始时间', '结束时间']} />
    </div>
  </div>
);

export default Demo;