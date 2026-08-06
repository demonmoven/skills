import { Tag } from '@coze-arch/coze-design';
import { IconCozFace } from '@coze-arch/coze-design/icons';

const Demo = () => (
  <div className="flex flex-col gap-4">
    <div>
      <h4>标准尺寸</h4>
      <div className="flex gap-2 mt-2">
        <Tag size="small" loading>
          加载中
        </Tag>
        <Tag size="small" disabled>
          已禁用
        </Tag>
        <Tag size="small" loading disabled>
          加载且禁用
        </Tag>
      </div>
    </div>
    <div>
      <h4>迷你尺寸</h4>
      <div className="flex gap-2 mt-2">
        <Tag size="mini" loading>
          加载中
        </Tag>
        <Tag size="mini" disabled>
          已禁用
        </Tag>
        <Tag size="mini" loading disabled>
          加载且禁用
        </Tag>
      </div>
    </div>
  </div>
);

export default Demo;