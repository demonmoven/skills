import { TimePicker } from '@coze-arch/coze-design';

const Demo = () => (
  <div className="flex flex-col gap-4">
    <div>
      <h4>禁用状态</h4>
      <TimePicker disabled placeholder="禁用状态" />
    </div>
    <div>
      <h4>禁用时间范围</h4>
      <TimePicker
        type="timeRange"
        disabled
        placeholder={['开始时间', '结束时间']}
      />
    </div>
  </div>
);

export default Demo;