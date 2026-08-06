import { InputCode } from '@coze-arch/coze-design';

const Demo = () => {
  return (
    <div>
      <InputCode
        type="text"
        onChange={value => console.log('当前输入：', value)}
        onFinish={value => console.log('输入完成：', value)}
      />
    </div>
  );
};

export default Demo;