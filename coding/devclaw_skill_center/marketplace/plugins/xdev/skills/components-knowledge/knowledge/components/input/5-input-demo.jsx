import { Input } from '@coze-arch/coze-design';
import { IconCozLoading } from '@coze-arch/coze-design/icons';

const Demo = () => {
  return (
    <div className="flex flex-col gap-4">
      <Input prefix="http://" suffix=".com" />
      <Input prefix={<IconCozLoading />} suffix={<IconCozLoading />} />
    </div>
  );
};

export default Demo;