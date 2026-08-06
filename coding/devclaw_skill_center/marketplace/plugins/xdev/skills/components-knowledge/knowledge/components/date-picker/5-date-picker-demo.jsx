import { DatePicker } from '@coze-arch/coze-design';

const Demo = () => {
  return (
    <div className="flex gap-4">
      <DatePicker showPrefix={false} placeholder="无前缀图标" />
      <DatePicker showSuffix={false} placeholder="无后缀图标" />
      <DatePicker showClear={false} placeholder="无清除按钮" />
    </div>
  );
};

export default Demo;