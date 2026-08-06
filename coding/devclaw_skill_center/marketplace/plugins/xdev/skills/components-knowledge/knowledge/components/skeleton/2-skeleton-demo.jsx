import React from 'react';
import { Skeleton } from '@coze-arch/coze-design';

const Demo = () => {
    const placeholder = (
        <div>
            <Skeleton.Image style={{ width: 200, height: 150 }} />
            <Skeleton.Title style={{ width: 120, marginTop: 10 }} />
        </div>
    );

    return (
        <Skeleton placeholder={placeholder} loading={true}>
            <img
                src="https://lf3-static.bytednsdoc.com/obj/eden-cn/ptlz_zlp/ljhwZthlaukjlkulzlp/root-web-sites/dy.png"
                height="150"
                alt="avatar"
            />
            <h4>Semi UI</h4>
        </Skeleton>
    );
};

export default Demo;
