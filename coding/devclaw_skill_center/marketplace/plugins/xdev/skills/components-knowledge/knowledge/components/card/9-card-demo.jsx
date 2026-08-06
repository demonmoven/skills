import React, { useState } from 'react';
import { Card, Switch } from '@coze-arch/coze-design';

const Demo = () => {
    const [loading, setLoading] = useState(true);
    const { Meta } = Card;

    return (
        <>
            <Switch onChange={ v => setLoading(!v) } />
            <br />
            <br />
            <Card 
                style={{ maxWidth: 360 }}
                loading={ loading }
            >
                <Meta 
                    title="Semi Doc" 
                    description="全面、易用、优质"
                />
            </Card>
        </>
    );
};

export default Demo;
