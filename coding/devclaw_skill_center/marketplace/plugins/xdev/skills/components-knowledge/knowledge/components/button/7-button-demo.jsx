import { Button, AIButton } from '@coze-arch/coze-design';

const Demo = () => (
  <div style={{ display: 'flex', gap: 8, flexWrap: 'wrap' }}>
    <Button color="brand">品牌色</Button>
    <Button color="hgltplus">高亮增强</Button>
    <Button color="highlight">高亮</Button>
    <Button color="primary">主色</Button>
    <Button color="secondary">次要</Button>
    <Button color="red">红色</Button>
    <Button color="redhglt">红色高亮</Button>
    <Button color="green">绿色</Button>
    <Button color="yellow">黄色</Button>
    <AIButton color="aiplus">AI增强</AIButton>
    <AIButton color="aihglt">AI高亮</AIButton>
    <AIButton color="aiprimary">AI主色</AIButton>
  </div>
);

export default Demo;