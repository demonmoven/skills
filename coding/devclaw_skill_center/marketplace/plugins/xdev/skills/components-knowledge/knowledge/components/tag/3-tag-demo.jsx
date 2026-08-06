import { Tag } from '@coze-arch/coze-design';
import { IconCozFace } from '@coze-arch/coze-design/icons';

const Demo = () => (
  <div className="flex flex-col gap-4">
    <div>
      <h4>标准尺寸</h4>
      <div className="flex gap-2 mt-2">
        <Tag size="small" prefixIcon="info">
          信息
        </Tag>
        <Tag size="small" prefixIcon="clock">
          等待中
        </Tag>
        <Tag size="small" prefixIcon="check">
          已完成
        </Tag>
        <Tag size="small" prefixIcon={<IconCozFace />}>
          自定义图标
        </Tag>
        <Tag size="small" prefixIcon="info" />
      </div>
    </div>
    <div>
      <h4>迷你尺寸</h4>
      <div className="flex gap-2 mt-2">
        <Tag size="mini" prefixIcon="info">
          信息
        </Tag>
        <Tag size="mini" prefixIcon="clock">
          等待中
        </Tag>
        <Tag size="mini" prefixIcon="check">
          已完成
        </Tag>
        <Tag size="mini" prefixIcon={<IconCozFace />}>
          自定义图标
        </Tag>
        <Tag size="mini" prefixIcon="info" />
      </div>
    </div>
  </div>
);

export default Demo;