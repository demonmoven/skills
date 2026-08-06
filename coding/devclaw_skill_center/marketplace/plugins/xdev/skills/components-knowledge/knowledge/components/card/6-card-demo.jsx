import React from 'react';
import { Card, Avatar, Space, Button, Typography } from '@coze-arch/coze-design';

const Demo = () => {
    const { Meta } = Card;
    const { Text } = Typography;

    return (
        <Card
            style={{ maxWidth: 340 }}
            title={
                <Meta 
                    title="Semi Doc" 
                    description="全面、易用、优质" 
                    avatar={
                        <Avatar 
                            alt='Card meta img'
                            size="default"
                            src='https://lf3-static.bytednsdoc.com/obj/eden-cn/ptlz_zlp/ljhwZthlaukjlkulzlp/card-meta-avatar-docs-demo.jpg'
                        />
                    }
                />
            }
            headerExtraContent={
                <Text link>
                    More
                </Text>
            }
            cover={ 
                <img 
                    alt="example" 
                    src="https://lf3-static.bytednsdoc.com/obj/eden-cn/ptlz_zlp/ljhwZthlaukjlkulzlp/root-web-sites/card-cover-docs-demo.jpeg" 
                />
            }
            footerLine={ true }
            footerStyle={{ display: 'flex', justifyContent: 'flex-end' }}
            footer={
                <Space>
                    <Button theme='borderless' type='primary'>精选案例</Button>
                    <Button theme='solid' type='primary'>开始使用</Button>
                </Space>
            }
        >
            Semi Design 是由抖音前端团队与 UED 团队共同设计开发并维护的设计系统。
        </Card>
    );
};

export default Demo;
