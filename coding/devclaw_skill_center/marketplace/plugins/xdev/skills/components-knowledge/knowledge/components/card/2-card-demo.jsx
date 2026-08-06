import React from 'react';
import { Card, Popover, Avatar } from '@coze-arch/coze-design';
import { IconCozInfoCircle } from '@coze-arch/coze-design/icons';

const Demo = () => {
    const { Meta } = Card;

    return (
        <>
            <Card style={{ maxWidth: 360 }} >
                Semi Design 是由抖音前端团队与 UED 团队共同设计开发并维护的设计系统。
            </Card>
            <br />
            <Card 
                style={{ maxWidth: 360 }} 
                bodyStyle={{ 
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'space-between'
                }}
            >
                <Meta 
                    title="Semi Doc" 
                    avatar={
                        <Avatar 
                            alt='Card meta img'
                            size="default"
                            src='https://lf3-static.bytednsdoc.com/obj/eden-cn/ptlz_zlp/ljhwZthlaukjlkulzlp/card-meta-avatar-docs-demo.jpg'
                        />
                    }
                />
                <Popover
                    position='top'
                    showArrow
                    content={
                        <article style={{ padding: 6 }}>
                            这是一个 Card
                        </article>
                    }
                >
                    <IconCozInfoCircle style={{ color: 'var(--semi-color-primary)' }}/>
                </Popover>
            </Card>
        </>
    );
};

export default Demo;
