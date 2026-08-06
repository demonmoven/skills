import { CozAvatar } from '@coze-arch/coze-design';

const Demo = () => {
  return (
    <div style={{ display: 'flex', gap: 12, alignItems: 'center' }}>
      <CozAvatar size="ultra">U</CozAvatar>
      <CozAvatar size="xxl">X</CozAvatar>
      <CozAvatar size="xl">X</CozAvatar>
      <CozAvatar size="lg">L</CozAvatar>
      <CozAvatar size="plus">P</CozAvatar>
      <CozAvatar size="default">D</CozAvatar>
      <CozAvatar size="small">S</CozAvatar>
      <CozAvatar size="mini">M</CozAvatar>
      <CozAvatar size="micro">M</CozAvatar>
    </div>
  );
};

export default Demo;