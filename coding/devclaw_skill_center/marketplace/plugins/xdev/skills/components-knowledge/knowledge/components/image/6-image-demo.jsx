import React, { useMemo, useCallback, useState } from 'react';
import { ImagePreview, Button } from '@coze-arch/coze-design';

const Demo = () => {
    const srcList = useMemo(() => ([
        "https://lf3-static.bytednsdoc.com/obj/eden-cn/ptlz_zlp/ljhwZthlaukjlkulzlp/root-web-sites/abstract.jpg",
        "https://lf3-static.bytednsdoc.com/obj/eden-cn/ptlz_zlp/ljhwZthlaukjlkulzlp/root-web-sites/sky.jpg",
        "https://lf3-static.bytednsdoc.com/obj/eden-cn/ptlz_zlp/ljhwZthlaukjlkulzlp/root-web-sites/greenleaf.jpg",
        "https://lf3-static.bytednsdoc.com/obj/eden-cn/ptlz_zlp/ljhwZthlaukjlkulzlp/root-web-sites/colorful.jpg",
    ]), []);

    const [visible1, setVisible1] = useState(false);
    const [visible2, setVisible2] = useState(false);

    const visibleChange1 = useCallback((v) => {
        setVisible1(v);
    }, []);

    const visibleChange2 = useCallback((v) => {
        setVisible2(v);
    }, []);

    const onButton1Click = useCallback((v) => {
        setVisible1(true);
    }, []);

    const onButton2Click = useCallback((v) => {
        setVisible2(true);
    }, []);

    return ( 
        <>
            <Button onClick={onButton1Click}>Preview single Image</Button>
            <ImagePreview
                src={"https://lf3-static.bytednsdoc.com/obj/eden-cn/ptlz_zlp/ljhwZthlaukjlkulzlp/root-web-sites/sky.jpg"}
                visible={visible1}
                onVisibleChange={visibleChange1}
            />
            <br /> 
            <Button onClick={onButton2Click} style={{ marginTop: 20 }}>Preview multiple Images</Button>
            <ImagePreview
                src={srcList}
                visible={visible2}
                onVisibleChange={visibleChange2}
            />
        </>
    );
};

export default Demo;
