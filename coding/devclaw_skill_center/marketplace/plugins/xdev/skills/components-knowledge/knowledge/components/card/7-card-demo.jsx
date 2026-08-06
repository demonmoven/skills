import React from 'react';
import { Card, Typography } from '@coze-arch/coze-design';

const Demo = () => {
    const { Text } = Typography;
  
    return (
        <Card title='Card title' >
            <Card 
                title='Inner Card title'
                style={{ marginBottom: 20 }}
                headerExtraContent={
                    <Text link>
                        More
                    </Text>
                }
            >
                Inner Card content
            </Card>
            <Card 
                title='Inner Card title'
                headerExtraContent={
                    <Text link>
                        More
                    </Text>
                }
            >
                Inner Card content
            </Card>
        </Card>
    );
};

export default Demo;
