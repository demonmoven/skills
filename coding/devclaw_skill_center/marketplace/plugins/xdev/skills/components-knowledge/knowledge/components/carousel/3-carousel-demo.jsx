import React, { useState } from 'react';
import { Carousel, Radio, Space, Typography } from '@coze-arch/coze-design';

const Demo = () => {
    const { Title, Paragraph } = Typography;
    const [size, setSize] = useState('small');
    const [type, setType] = useState('dot');
    const [position, setPosition] = useState('left');

    const style = {
        width: '100%',
        height: '400px',
    };

    const titleStyle = { 
        position: 'absolute', 
        top: '100px', 
        left: '100px'
    };

    const colorStyle = {
        color: '#1C1F23'
    };

    const renderLogo = () => {
        return (
            <img src='https://lf3-static.bytednsdoc.com/obj/eden-cn/ptlz_zlp/ljhwZthlaukjlkulzlp/root-web-sites/semi_logo.svg' alt='semi_logo' style={{ width: 87, height: 31 }}/>
        );
    };

    const imgList = [
        'https://lf3-static.bytednsdoc.com/obj/eden-cn/hjeh7pldnulm/SemiDocs/bg-1.png',
        'https://lf3-static.bytednsdoc.com/obj/eden-cn/hjeh7pldnulm/SemiDocs/bg-2.png',
        'https://lf3-static.bytednsdoc.com/obj/eden-cn/hjeh7pldnulm/SemiDocs/bg-3.png',
    ];

    const textList = [
        ['Semi 设计管理系统', '从 Semi Design,到 Any Design', '快速定制你的设计系统,并应用在设计稿和代码中'],
        ['Semi 物料市场', '面向业务场景的定制化组件,支持线上预览和调试', '内容由 Semi Design 用户共建'],
        ['Semi 设计/代码模板', '高效的 Design2Code 设计稿转代码', '海量 Figma前端代码一键转'],
    ];

    return (
        <div>
            <Carousel style={style} indicatorType={type} indicatorPosition={position} indicatorSize={size} theme='dark' autoPlay={false}>
                {
                    imgList.map((src, index) => {
                        return (
                            <div key={index} style={{ backgroundSize: 'cover', backgroundImage: `url('${src}')` }}>
                                <Space vertical align='start' spacing='medium' style={titleStyle}>
                                    {renderLogo()}
                                    <Title heading={2} style={colorStyle}>{textList[index][0]}</Title>
                                    <Space vertical align='start'>
                                        <Paragraph style={colorStyle}>{textList[index][1]}</Paragraph>
                                        <Paragraph style={colorStyle}>{textList[index][2]}</Paragraph>
                                    </Space>
                                </Space>
                            </div>
                        );
                    })
                }
            </Carousel>
            <br/>
            <Space vertical align='start'>
                <Space> 
                    <div>类型</div>
                    <Radio.Group onChange={e => setType(e.target.value)} value={type} type="button">
                        <Radio value='dot'>dot</Radio>
                        <Radio value='line'>line</Radio>
                        <Radio value='columnar'>columnar</Radio>
                    </Radio.Group>
                </Space>
                <Space> 
                    <div>位置</div>
                    <Radio.Group onChange={e => setPosition(e.target.value)} value={position} type="button">
                        <Radio value='left'>left</Radio>
                        <Radio value='center'>center</Radio>
                        <Radio value='right'>right</Radio>
                    </Radio.Group>
                </Space>
                <Space> 
                    <div>尺寸</div>
                    <Radio.Group onChange={e => setSize(e.target.value)} value={size} type="button">
                        <Radio value='small'>small</Radio>
                        <Radio value='medium'>medium</Radio>
                    </Radio.Group>
                </Space>
            </Space>
        </div>
    );
};

export default Demo;
