import { TextArea } from '@coze-arch/coze-design';

const Demo = () => (
  <div className="flex flex-col gap-4">
    <TextArea placeholder="请输入内容" />
    <TextArea defaultValue="这是默认内容" />
    <TextArea placeholder="最多输入 100 个字符" maxCount={100} />
  </div>
);

export default Demo;