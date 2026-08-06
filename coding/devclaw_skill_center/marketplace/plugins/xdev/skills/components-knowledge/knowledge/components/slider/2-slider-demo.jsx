import React from 'react';
import { Slider, InputNumber } from '@coze-arch/coze-design';

class Demo extends React.Component {
    constructor(props) {
        super();
        this.state = { value: 10 };
        this.getSliderValue = this.getSliderValue.bind(this);
    }

    getSliderValue(value) {
        if (isNaN(Number(value))){
            return;
        }
        this.setState({ value: value / 1 }); 
    }

    render() {
        const { value } = this.state;
        return (
            <div>
                <div style={{ width: 320, marginRight: 15 }}>
                    <Slider step={1} value={value} onChange={(value) => (this.getSliderValue(value))} ></Slider>
                </div>
                <InputNumber onChange={(v) => this.getSliderValue(v)} style={{ width: 100 }} value={value} min={0} max={100} />
            </div>
        );
    }
}

export default Demo;
