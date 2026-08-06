import { CozAvatar, Badge } from '@coze-arch/coze-design';

const Demo = () => (
  <div className="flex flex-row gap-4 p-4">
    <Badge type="mini" position="rightTop" countStyle={{ right: 8, top: 6 }}>
      <CozAvatar color="green" type="person" size="lg">
        BD
      </CozAvatar>
    </Badge>
    <Badge
      type="mini"
      position="rightBottom"
      countStyle={{ right: 8, bottom: 6 }}
    >
      <CozAvatar color="green" type="person" size="lg">
        BD
      </CozAvatar>
    </Badge>
    <Badge type="mini" position="leftTop" countStyle={{ left: 8, top: 6 }}>
      <CozAvatar color="green" type="person" size="lg">
        BD
      </CozAvatar>
    </Badge>
    <Badge
      type="mini"
      position="leftBottom"
      countStyle={{ left: 8, bottom: 6 }}
    >
      <CozAvatar color="green" type="person" size="lg">
        BD
      </CozAvatar>
    </Badge>
  </div>
);

export default Demo;