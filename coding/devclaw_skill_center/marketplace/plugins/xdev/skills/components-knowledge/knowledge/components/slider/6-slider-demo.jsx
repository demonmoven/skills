import React from 'react';
import { Slider, Button } from '@coze-arch/coze-design';

class Demo extends React.Component {
    constructor(props) {
        super();
        this.state = { value: 10 };
        this.changeValue = this.changeValue.bind(this);
    }

    changeValue() {
        this.setState({ value: this.state.value + 10 });
    }

    render() {
        return (
            <div>
                <Button onClick={this.changeValue} style={{ marginRight: 20 }}>点击改变value值</Button>
                <br/>
                <br/>
                <Slider value={this.state.value}></Slider>
            </div>
        );
    }
}

export default Demo;
