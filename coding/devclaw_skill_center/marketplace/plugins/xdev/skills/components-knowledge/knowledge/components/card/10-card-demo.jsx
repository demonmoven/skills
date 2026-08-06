import React, { useState } from 'react';
import { Card, Switch, Skeleton, Avatar, Typography } from '@coze-arch/coze-design';

const Demo = () => {
    const [loading, setLoading] = useState(true);
    const { Meta } = Card;
    const { Title, Paragraph, Image } = Skeleton;

    return (
        <>
            <Switch onChange={ v => setLoading(!v) } />
            <br />
            <br />
            <Card
                style={{ maxWidth: 300 }}
                title={
                    <Meta 
                        title={
                            <Skeleton
                                style={{ width: 80 }}
                                placeholder={<Title />}
                                loading={loading}
                            >
                                <Typography.Title heading={5}>
                                    Semi Doc
                                </Typography.Title>
                            </Skeleton>
                        } 
                        description={
                            <Skeleton 
                                style={{ width: 150, marginTop: 12 }} 
                                placeholder={<Paragraph rows={1} />} 
                                loading={loading}
                            >
                                <Typography.Text>
                                    全面、易用、优质
                                </Typography.Text>
                            </Skeleton>
                        }
                        avatar={
                            <Skeleton placeholder={<Skeleton.Avatar />} loading={loading}>
                                <Avatar 
                                    alt='Card meta img'
                                    size="default"
                                    src='https://lf3-static.bytednsdoc.com/obj/eden-cn/ptlz_zlp/ljhwZthlaukjlkulzlp/card-meta-avatar-docs-demo.jpg'
                                />
                            </Skeleton>
                        }
                    />
                }
                headerExtraContent={
                    <Skeleton style={{ width: 50 }} placeholder={<Paragraph rows={1} />} loading={loading}>
                        <Typography.Text link>
                            More
                        </Typography.Text>
                    </Skeleton>
                }
                cover={ 
                    <Skeleton style={{ maxWidth: '100%', height: 220 }} placeholder={<Image />} loading={loading}>
                        <img 
                            alt="example" 
                            src="https://lf3-static.bytednsdoc.com/obj/eden-cn/ptlz_zlp/ljhwZthlaukjlkulzlp/root-web-sites/card-cover-docs-demo.jpeg" 
                        />
                    </Skeleton> 
                }
            >
            </Card>
        </>
    );
};

export default Demo;
