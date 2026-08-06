import { CozInputNumber } from '@coze-arch/coze-design';

const Demo = () => {
  return (
    <CozInputNumber
      defaultValue={1}
      min={0}
      max={100}
      onChange={value => console.log('当前值：', value)}
    />
  );
};

export default Demo;