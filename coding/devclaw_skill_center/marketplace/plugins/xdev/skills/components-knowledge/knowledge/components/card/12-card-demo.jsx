import React from 'react';
import { Card, Rating } from '@coze-arch/coze-design';

const Demo = () => {
    const { Meta } = Card;

    return (
        <Card
            style={{ maxWidth: 300 }}
            actions={[    
                // eslint-disable-next-line react/jsx-key
                <Rating size='small' defaultValue={4}/>
            ]}
            headerLine={ false }
            cover={ 
                <img 
                    alt="example" 
                    src="https://lf3-static.bytednsdoc.com/obj/eden-cn/ptlz_zlp/ljhwZthlaukjlkulzlp/root-web-sites/card-cover-docs-demo.jpeg" 
                />
            }
        >
            <Meta 
                title="Semi Doc" 
                description="全面、易用、优质" 
            />
        </Card>
    );
};

export default Demo;
