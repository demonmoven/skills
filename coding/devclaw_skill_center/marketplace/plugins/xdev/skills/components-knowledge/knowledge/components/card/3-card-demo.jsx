import React from 'react';
import { Card } from '@coze-arch/coze-design';

const Demo = () => {
    const { Meta } = Card;

    return (
        <Card
            style={{ maxWidth: 300 }}
            cover={ 
                <img 
                    alt="example" 
                    src="https://lf3-static.bytednsdoc.com/obj/eden-cn/ptlz_zlp/ljhwZthlaukjlkulzlp/root-web-sites/card-cover-docs-demo2.jpeg" 
                />
            }
        >
            <Meta title="卡片封面" />
        </Card>
    );
};

export default Demo;
