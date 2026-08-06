import { TextArea } from '@coze-arch/coze-design';

const Demo = () => (
  <div className="flex flex-col gap-4">
    <div>
      <h4>禁用状态</h4>
      <TextArea disabled placeholder="禁用状态" />
    </div>
    <div>
      <h4>加载状态</h4>
      <TextArea loading placeholder="加载中..." />
    </div>
    <div>
      <h4>错误状态</h4>
      <TextArea error placeholder="错误状态" />
    </div>
  </div>
);

export default Demo;