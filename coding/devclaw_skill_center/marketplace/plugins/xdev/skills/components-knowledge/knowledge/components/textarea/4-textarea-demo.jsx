import { TextArea } from '@coze-arch/coze-design';
import { IconCozLoading } from '@coze-arch/coze-design/icons';

const Demo = () => (
  <div className="flex flex-col gap-4">
    <div>
      <h4>前缀图标</h4>
      <TextArea prefix={<IconCozLoading />} placeholder="带前缀图标" />
    </div>
    <div>
      <h4>后缀图标</h4>
      <TextArea suffix={<IconCozLoading />} placeholder="带后缀图标" />
    </div>
  </div>
);

export default Demo;