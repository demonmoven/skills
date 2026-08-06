import { Banner, Button } from '@coze-arch/coze-design';

const Demo = () => (
  <div className="border border-solid coz-stroke-primary rounded w-480px p-20px">
    <Banner
      card={true}
      fullMode={false}
      title="标题"
      type="info"
      bordered={false}
      description="这是一条卡片样式的横幅提示信息"
    >
      <div className="text-right px-12px my-12px">
        <Button color="hgltplus">不需要</Button>
      </div>
    </Banner>
  </div>
);

export default Demo;