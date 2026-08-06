import React from 'react';
import { Slider } from '@coze-arch/coze-design';

class Demo extends React.Component {
    constructor(props) {
        super();
        this.state = { value: [20, 60] };
        this.changeValue = this.changeValue.bind(this);
        this.getRailStyle = this.getRailStyle.bind(this);
    }

    changeValue(value) {
        this.setState({ value });
    }

    getRailStyle(range) {
    // color of second segment inherits from .semi-slider-track
        const color = ['var(--semi-color-danger)', 'transparent', 'var(--semi-color-success)'];
        const gradientPos = this.state.value.map(val => 
            ((val - range[0]) / (range[1] - range[0])).toFixed(2) * 100
        );
        const style = {
            background: `linear-gradient(to right, ${color[0]} ${gradientPos[0]}%, ${color[1]} ${gradientPos[0]}%, ${color[1]} ${gradientPos[1]}%, ${color[2]} ${gradientPos[1]}%)`
        };
        return style;
    }

    render() {
        const range = [10, 100];
        const railStyle = this.getRailStyle(range);
        return (
            <Slider
                range
                min={range[0]}
                max={range[1]}
                onChange={this.changeValue}
                railStyle={railStyle}
                defaultValue={this.state.value}
            />
        );
    }
}

export default Demo;
