import { Input } from '@coze-arch/coze-design';

const Demo = () => {
  return (
    <div className="flex flex-col gap-4">
      <Input size="small" placeholder="小尺寸" />
      <Input placeholder="默认尺寸" />
    </div>
  );
};

export default Demo;