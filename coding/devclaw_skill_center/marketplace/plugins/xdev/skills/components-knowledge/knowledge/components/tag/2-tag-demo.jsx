import { Tag } from '@coze-arch/coze-design';
import { IconCozFace } from '@coze-arch/coze-design/icons';

const colors = [
  'brand',
  'primary',
  'green',
  'yellow',
  'red',
  'cyan',
  'blue',
  'purple',
  'magenta',
] as const;

const Demo = () => (
  <div className="flex flex-col gap-4">
    <div>
      <h4>普通状态</h4>
      <div className="flex gap-2 mt-2">
        {colors.map(color => (
          <Tag key={color} color={color}>
            {color}
          </Tag>
        ))}
      </div>
    </div>
    <div>
      <h4>可交互</h4>
      <div className="flex gap-2 mt-2">
        {colors.map(color => (
          <Tag
            key={color}
            color={color}
            onClick={() => console.log('clicked:', color)}
          >
            {color}
          </Tag>
        ))}
      </div>
    </div>
    <div>
      <h4>禁用状态</h4>
      <div className="flex gap-2 mt-2">
        {colors.map(color => (
          <Tag key={color} color={color} disabled>
            {color}
          </Tag>
        ))}
      </div>
    </div>
  </div>
);

export default Demo;