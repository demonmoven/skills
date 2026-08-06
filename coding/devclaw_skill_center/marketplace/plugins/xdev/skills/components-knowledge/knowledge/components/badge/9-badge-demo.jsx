import { Badge, Button, IconButton } from '@coze-arch/coze-design';
import {
  IconCozPeopleFill,
  IconCozMicrophone,
  IconCozBell,
} from '@coze-arch/coze-design/icons';

const Demo = () => (
  <div className="flex flex-col gap-6 p-4">
    <div>
      <h4 className="mb-2 text-sm">与普通按钮结合：</h4>
      <div className="flex gap-4">
        <Button icon={<IconCozPeopleFill />} showBadge={true}>
          按钮
        </Button>
        <Button
          icon={<IconCozPeopleFill />}
          showBadge={true}
          badgeColor="unset"
          color="highlight"
        >
          高亮按钮
        </Button>
        <Button
          icon={<IconCozBell />}
          showBadge={true}
          badgeCount={5}
          color="primary"
        >
          消息按钮
        </Button>
      </div>
    </div>

    <div>
      <h4 className="mb-2 text-sm">与图标按钮结合：</h4>
      <div className="flex gap-4 items-center">
        <Badge type="mini">
          <IconButton size="large" icon={<IconCozMicrophone />} />
        </Badge>

        <Badge count={9}>
          <IconButton icon={<IconCozMicrophone />} />
        </Badge>

        <Badge type="mini">
          <IconButton
            size="small"
            icon={<IconCozMicrophone />}
            color="secondary"
          />
        </Badge>

        <Badge count={99} overflowCount={99}>
          <IconButton size="large" icon={<IconCozBell />} color="primary" />
        </Badge>
      </div>
    </div>
  </div>
);

export default Demo;