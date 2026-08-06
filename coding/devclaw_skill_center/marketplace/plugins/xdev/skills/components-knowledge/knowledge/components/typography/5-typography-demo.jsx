import { Typography } from '@coze-arch/coze-design';

const { Numeral } = Typography;

const Demo = () => (
  <div className="flex flex-col gap-2">
    <Numeral className="coz-fg-primary" precision={2}>
      <p>点赞量：1.6111e1 K</p>
    </Numeral>
    <Numeral rule="percentages" className="coz-fg-primary" precision={2}>
      <p>好评率: 0.915</p>
    </Numeral>
  </div>
);

export default Demo;