import { DatePicker } from '@coze-arch/coze-design';

const Demo = () => {
  return (
    <div className="flex gap-4">
      <DatePicker size="default" placeholder="默认尺寸" />
      <DatePicker size="small" placeholder="小尺寸" />
    </div>
  );
};

export default Demo;