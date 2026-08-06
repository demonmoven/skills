import { Loading } from '@coze-arch/coze-design';

const Demo = () => {
  return (
    <div className="flex space-x-8 p-20px gap-6">
      <Loading loading size="mini" />
      <Loading loading size="small" />
      <Loading loading size="middle" />
      <Loading loading size="large" />
    </div>
  );
};

export default Demo;