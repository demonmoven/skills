import React, { useMemo, useCallback } from 'react';
import { Image, ImagePreview, Divider, Tooltip } from '@coze-arch/coze-design';
import { IconCozInfoCircle } from '@coze-arch/coze-design/icons';

const Demo = () => {
    const srcList = useMemo(() => ([
        "https://lf3-static.bytednsdoc.com/obj/eden-cn/ptlz_zlp/ljhwZthlaukjlkulzlp/root-web-sites/abstract.jpg",
        "https://lf3-static.bytednsdoc.com/obj/eden-cn/ptlz_zlp/ljhwZthlaukjlkulzlp/root-web-sites/sky.jpg",
        "https://lf3-static.bytednsdoc.com/obj/eden-cn/ptlz_zlp/ljhwZthlaukjlkulzlp/root-web-sites/greenleaf.jpg",
    ]), []);

    const renderPreviewMenu = useCallback((props) => {
        const { menuItems } = props;
        const customNode = <Tooltip content='我是一个自定义操作'><IconCozInfoCircle size="large" /></Tooltip>;
        return (
            <div style={{ display: 'flex', backgroundColor: 'rgba(0, 0, 0, 0.75)', alignItems: 'center', padding: '5px 16px', borderRadius: 4 }}>
                {menuItems.slice(0, 3)}
                <Divider layout="vertical" />
                {menuItems.slice(3, 7)}
                <Divider layout="vertical" />
                {menuItems.slice(7)}
                <Divider layout="vertical" />
                {customNode}
            </div>
        );
    }, []);

    return (
        <>  
            <ImagePreview
                renderPreviewMenu={renderPreviewMenu}
            >
                {srcList.map((src, index) => (<Image key={index} src={src} width={200} alt={'lamp' + (index + 1)} />))}
            </ImagePreview>
        </>
    );
};

export default Demo;
