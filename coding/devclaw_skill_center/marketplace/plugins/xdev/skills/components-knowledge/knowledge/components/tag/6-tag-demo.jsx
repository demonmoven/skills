import { Tag, Tooltip } from '@coze-arch/coze-design';
import { IconCozInfoCircle } from '@coze-arch/coze-design/icons';

const Demo = () => (
  <div className="flex flex-col gap-4">
    <div>
      <h4>信息图标提示</h4>
      <div className="flex gap-2 mt-2">
        <Tag
          size="small"
          color="blue"
          suffixIcon={
            <Tooltip content="该功能正在公测中，如有问题请及时反馈">
              <IconCozInfoCircle />
            </Tooltip>
          }
        >
          Beta 功能
        </Tag>
      </div>
    </div>
  </div>
);

export default Demo;