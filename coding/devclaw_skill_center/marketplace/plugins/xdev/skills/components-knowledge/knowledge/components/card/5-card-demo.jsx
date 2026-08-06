import React from 'react';
import { Card, Avatar, Popover } from '@coze-arch/coze-design';
import { IconCozInfoCircle } from '@coze-arch/coze-design/icons';

const Demo = () => {
    const { Meta } = Card;

    return (
        <div>
            <Card 
                shadows='hover'
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
            <br/>
            <Card 
                shadows='always'
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
        </div>
    );
};

export default Demo;
