import React from 'react';
import { Upload } from '@coze-arch/coze-design';
import { IconCozLightning } from '@coze-arch/coze-design/icons';

const Demo = () => (<Upload
    action="https://api.semi.design/upload"
    dragIcon={<IconCozLightning />}
    draggable={true}
    accept="application/pdf,.jpeg"
    style={{ marginTop: 10 }}
>
    <div className="components-upload-demo-drag-area">
        <img
            src="https://lf3-static.bytednsdoc.com/obj/eden-cn/ptlz_zlp/ljhwZthlaukjlkulzlp/root-web-sites/dy.png"
            height="96"
            alt='demo img'
            style={{ borderRadius: 4 }}
        />
        <div
            style={{
                fontSize: 14,
                marginTop: 8,
                flexBasis: '100%',
                textAlign: 'center',
                color: 'var(--semi-color-tertiary)',
            }}
        >
            Wow, you can really dance.
        </div>
    </div>
</Upload>
);

export default Demo;
