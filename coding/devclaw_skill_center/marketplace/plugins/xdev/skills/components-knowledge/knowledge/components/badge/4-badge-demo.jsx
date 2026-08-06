import { CozAvatar, Badge } from '@coze-arch/coze-design';
import { IconCozFireFill } from '@coze-arch/coze-design/icons';

const Demo = () => (
  <div className="flex flex-row gap-4 p-4">
    <Badge
      count={<IconCozFireFill className="coz-fg-hglt-red text-20px" />}
      countStyle={{ top: -3, right: 8 }}
    >
      <CozAvatar color="purple" type="person" size="lg">
        BD
      </CozAvatar>
    </Badge>
  </div>
);

export default Demo;