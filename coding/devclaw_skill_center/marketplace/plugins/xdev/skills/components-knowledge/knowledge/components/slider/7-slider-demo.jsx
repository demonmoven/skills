import React from 'react';
import { Slider } from '@coze-arch/coze-design';

const Demo = () => (
    <div>
        <div style={{ height: 300, marginLeft: 30, marginTop: 10, paddingRight: 30, display: 'inline-block' }}>
            <Slider vertical></Slider>
        </div>
        <div style={{ height: 300, marginLeft: 30, marginTop: 10, paddingRight: 30, display: 'inline-block' }}>
            <Slider vertical verticalReverse></Slider>
        </div>
        <div style={{ height: 300, marginLeft: 30, marginTop: 10, paddingRight: 30, display: 'inline-block' }}>
            <Slider vertical range defaultValue={[20, 60]}></Slider>
        </div>
        <div style={{ height: 300, marginLeft: 30, marginTop: 10, paddingRight: 30, display: 'inline-block' }}>
            <Slider vertical verticalReverse range defaultValue={[20, 60]}></Slider>
        </div>
        <div style={{ height: 300, marginLeft: 30, marginTop: 10, paddingRight: 30, display: 'inline-block' }}>
            <Slider vertical range marks={{ 20: '20°C', 40: '40°C' }} step={10} defaultValue={[20, 60]}></Slider>
        </div>
        <div style={{ height: 300, marginLeft: 30, marginTop: 10, paddingRight: 30, display: 'inline-block' }}>
            <Slider vertical verticalReverse range marks={{ 20: '20°C', 40: '40°C' }} step={10} defaultValue={[20, 60]}></Slider>
        </div>
    </div>
);

export default Demo;
