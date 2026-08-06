import React from 'react';
import { Image, Button } from '@coze-arch/coze-design';

const Demo = () => {
    const [timestamp, setTimestamp] = React.useState('');
    return (  
        <>
            <Image 
                width={300}
                height={200}
                src={`https://lf3-static.bytednsdoc.com/obj/eden-cn/ptlz_zlp/ljhwZthlaukjlkulzlp/root-web-sites/abstract-big.png?${timestamp}`}
                placeholder={<Image 
                    src='https://lf3-static.bytednsdoc.com/obj/eden-cn/ptlz_zlp/ljhwZthlaukjlkulzlp/root-web-sites/abstract-small.jpeg'
                    width={300}
                    height={200}
                    preview={false}
                />}
            />
            <br />
            <Button 
                theme={'solid'}
                onClick={() => {
                    setTimestamp(Date.now());
                }}
                style={{ marginTop: 10 }}
            >Reload</Button>
        </>
    );
};

export default Demo;
