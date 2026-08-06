import React from 'react';
import { Carousel, Typography, Space } from '@coze-arch/coze-design';
import { IconCozArrowLeft, IconCozArrowRight } from "@coze-arch/coze-design/icons";

class Demo extends React.Component {
    constructor(props) {
        super(props);
        this.imgList = [
            'https://lf3-static.bytednsdoc.com/obj/eden-cn/hjeh7pldnulm/SemiDocs/bg-1.png',
            'https://lf3-static.bytednsdoc.com/obj/eden-cn/hjeh7pldnulm/SemiDocs/bg-2.png',
            'https://lf3-static.bytednsdoc.com/obj/eden-cn/hjeh7pldnulm/SemiDocs/bg-3.png',
        ];
        this.textList = [
            ['Semi 设计管理系统', '从 Semi Design,到 Any Design', '快速定制你的设计系统,并应用在设计稿和代码中'],
            ['Semi 物料市场', '面向业务场景的定制化组件,支持线上预览和调试', '内容由 Semi Design 用户共建'],
            ['Semi 设计/代码模板', '高效的 Design2Code 设计稿转代码', '海量 Figma 设计模板一键转为真实前端代码'],
        ];
        this.arrowProps = {
            leftArrow: { children: <IconCozArrowLeft size='large'/> },
            rightArrow: { children: <IconCozArrowRight size='large'/> },
        };
    };

    renderLogo() {
        return (
            <img src='https://lf3-static.bytednsdoc.com/obj/eden-cn/ptlz_zlp/ljhwZthlaukjlkulzlp/root-web-sites/semi_logo.svg' alt='semi_logo' style={{ width: 87, height: 31 }} />
        );
    };

    render() {
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

        return (
            <div>
                <Carousel 
                    theme='dark'
                    style={style} 
                    autoPlay={false} 
                    arrowProps={this.arrowProps}
                >
                    {
                        this.imgList.map((src, index) => {
                            return (
                                <div key={index} style={{ backgroundSize: 'cover', backgroundImage: `url('${src}')` }}>
                                    <Space vertical align='start' spacing='medium' style={titleStyle}>
                                        {this.renderLogo()}
                                        <Typography.Title heading={2} style={colorStyle}>{this.textList[index][0]}</Typography.Title>
                                        <Space vertical align='start'>
                                            <Typography.Paragraph style={colorStyle}>{this.textList[index][1]}</Typography.Paragraph>
                                            <Typography.Paragraph style={colorStyle}>{this.textList[index][2]}</Typography.Paragraph>
                                        </Space>
                                    </Space>
                                </div>
                            );
                        })
                    }
                </Carousel>
            </div>
        );
    }
}

export default Demo;
