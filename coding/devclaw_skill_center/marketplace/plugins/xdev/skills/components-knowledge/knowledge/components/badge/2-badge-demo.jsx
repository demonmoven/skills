import { CozAvatar, Badge, AIButton } from '@coze-arch/coze-design';

const Demo = () => (
  <div className="flex flex-row gap-4 p-4">
    <Badge type="default" count={666} countStyle={{ right: 2, top: 2 }}>
      <CozAvatar color="green" type="bot" size="lg">
        BD
      </CozAvatar>
    </Badge>
    <Badge type="alt" count="beta">
      <AIButton color="aihglt">Bot竞技场</AIButton>
    </Badge>
    <Badge type="mini" countStyle={{ right: 8, top: 6 }}>
      <CozAvatar color="indigo" type="person" size="lg">
        BD
      </CozAvatar>
    </Badge>
  </div>
);

export default Demo;