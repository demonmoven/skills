import { DatePicker } from '@coze-arch/coze-design';

const Demo = () => {
  return (
    <div className="flex gap-4">
      <DatePicker disabled placeholder="禁用状态" />
      <DatePicker type="dateRange" disabled placeholder="禁用范围选择" />
    </div>
  );
};

export default Demo;