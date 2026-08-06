import { Loading } from '@coze-arch/coze-design';

const Demo = () => {
  return (
    <div className="flex space-x-8 p-20px gap-6">
      <Loading loading color="default" />
      <Loading loading color="blue" />
      <Loading loading color="green" />
      <Loading loading color="red" />
    </div>
  );
};

export default Demo;