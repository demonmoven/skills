import { Input } from '@coze-arch/coze-design';

const Demo = () => {
  return (
    <Input
      placeholder="输入中文时不会触发字数统计"
      maxLength={10}
      onCompositionStart={() => console.log('开始输入')}
      onCompositionEnd={() => console.log('输入完成')}
    />
  );
};

export default Demo;