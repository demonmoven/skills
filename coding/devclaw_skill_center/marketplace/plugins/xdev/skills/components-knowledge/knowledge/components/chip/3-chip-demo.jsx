import { Chip } from '@coze-arch/coze-design';

const Demo = () => (
  <div style={{ display: 'flex', gap: '8px', flexWrap: 'wrap' }}>
    <Chip color="brand">brand</Chip>
    <Chip color="primary">primary</Chip>
    <Chip color="green">green</Chip>
    <Chip color="yellow">yellow</Chip>
    <Chip color="red">red</Chip>
    <Chip color="cyan">cyan</Chip>
    <Chip color="blue">blue</Chip>
    <Chip color="purple">purple</Chip>
    <Chip color="magenta">magenta</Chip>
  </div>
);

export default Demo;