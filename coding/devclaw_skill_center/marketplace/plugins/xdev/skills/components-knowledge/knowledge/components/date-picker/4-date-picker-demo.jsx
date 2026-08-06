import { DatePicker } from '@coze-arch/coze-design';

const Demo = () => {
  return (
    <div className="flex gap-4">
      <DatePicker format="yyyy年MM月dd日" placeholder="自定义日期格式" />
      <DatePicker
        type="dateTime"
        format="yyyy年MM月dd日 hh时mm分"
        placeholder="自定义日期时间格式"
      />
    </div>
  );
};

export default Demo;