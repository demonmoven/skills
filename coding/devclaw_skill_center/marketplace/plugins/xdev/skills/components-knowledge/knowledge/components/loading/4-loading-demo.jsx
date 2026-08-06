import { Loading } from '@coze-arch/coze-design';

const Demo = () => {
  return (
    <div className="flex flex-col space-y-4 p-20px gap-6">
      <Loading loading label={<span>加载中...</span>} className="flex">
        <span />
      </Loading>
      <Loading
        loading
        label={<span>正在处理...</span>}
        labelSize="large"
        className="flex"
      >
        <span />
      </Loading>
    </div>
  );
};

export default Demo;