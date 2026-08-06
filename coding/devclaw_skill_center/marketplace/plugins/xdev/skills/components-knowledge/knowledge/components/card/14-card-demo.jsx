import React from 'react';
import { Card, CardGroup, Typography } from '@coze-arch/coze-design';

const Demo = () => {
    const { Text } = Typography;

    return (
        <CardGroup type='grid'>
            {
                new Array(7).fill(null).map((v, idx)=>(
                    <Card 
                        key={idx}
                        shadows='hover'
                        title='Card title'
                        headerLine={false}
                        style={{ width: 260 }}
                        headerExtraContent={
                            <Text link>
                                More
                            </Text>
                        }
                    >
                        <Text>Card content</Text>
                    </Card>
                ))
            }     
        </CardGroup>
    );
};

export default Demo;
