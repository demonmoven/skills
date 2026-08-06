import { Input } from '@coze-arch/coze-design';

const Demo = () => {
  return (
    <div className="flex flex-col gap-4">
      <Input maxLength={10} placeholder="最多输入10个字符" />
      <Input
        maxLength={10}
        getValueLength={value => value.replace(/[\u4e00-\u9fa5]/g, 'aa').length}
        placeholder="中文字符按2个长度计算"
      />
    </div>
  );
};

export default Demo;