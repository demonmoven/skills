import { Badge, CozAvatar } from '@coze-arch/coze-design';

const Demo = () => (
  <div className="flex gap-4 p-4">
    <Badge
      count={8}
      countStyle={{
        backgroundColor: 'var(--coz-fg-hglt-green)',
        color: 'white',
        fontWeight: 'bold',
      }}
    >
      <CozAvatar color="orange" type="bot" size="lg">
        BD
      </CozAvatar>
    </Badge>

    <Badge
      count="HOT"
      type="alt"
      countStyle={{
        backgroundColor: 'var(--coz-fg-hglt-red)',
        padding: '0 8px',
      }}
    >
      <CozAvatar color="purple" type="bot" size="lg">
        BD
      </CozAvatar>
    </Badge>

    <Badge
      type="mini"
      countStyle={{
        backgroundColor: 'var(--coz-fg-hglt-yellow)',
        width: '12px',
        height: '12px',
        right: 5,
        top: 5,
      }}
    >
      <CozAvatar color="green" type="bot" size="lg">
        BD
      </CozAvatar>
    </Badge>
  </div>
);

export default Demo;