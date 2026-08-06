import { DatePicker } from '@coze-arch/coze-design';

const Demo = () => {
  return (
    <div className="flex flex-col gap-4">
      <div className="flex gap-4">
        <DatePicker placeholder="选择日期" />
        <DatePicker type="dateRange" placeholder="选择日期范围" />
      </div>
      <div className="flex gap-4">
        <DatePicker type="month" placeholder="选择月份" />
        <DatePicker type="monthRange" placeholder="选择月份范围" />
      </div>
      <div className="flex gap-4">
        <DatePicker type="dateTime" placeholder="选择日期时间" />
        <DatePicker type="dateTimeRange" placeholder="选择日期时间范围" />
      </div>
    </div>
  );
};

export default Demo;