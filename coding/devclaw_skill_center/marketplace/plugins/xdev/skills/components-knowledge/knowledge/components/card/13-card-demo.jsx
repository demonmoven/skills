import React, { useState } from 'react';
import { Card, CardGroup, Typography, Slider } from '@coze-arch/coze-design';

const Demo = () => {
    const { Text } = Typography;
    const [ spacing, setSpacing ] = useState(12);

    return (
        <>
            <Text>滑动调节 Card 间距</Text>
            <Slider 
                defaultValue={12}
                max={40}
                min={10}
                style={{ width: 360 }}
                onChange={v=> setSpacing(v)}
            />
            <br />
            <CardGroup spacing={spacing}>
                {
                    new Array(8).fill(null).map((v, idx)=>(
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
        </>
    );
};

export default Demo;
