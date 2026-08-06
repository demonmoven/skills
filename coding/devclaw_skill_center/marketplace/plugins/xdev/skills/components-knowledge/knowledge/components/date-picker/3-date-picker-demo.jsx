import { DatePicker } from '@coze-arch/coze-design';

const Demo = () => {
  return (
    <div className="flex gap-4">
      <DatePicker defaultValue="2024-03-07" placeholder="默认值" />
      <DatePicker multiple placeholder="多选模式" />
    </div>
  );
};

export default Demo;