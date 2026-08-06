import React, { useMemo, useCallback } from 'react';
import { Image, ImagePreview, Button } from '@coze-arch/coze-design';
import { IconCozArrowLeft, IconCozArrowRight, IconCozMinus, IconCozPlus, IconCozRotationFill, IconCozDownload, IconCozOriginalSize, IconCozAutoView } from "@coze-arch/coze-design/icons";

const Demo = () => {
    const srcList = useMemo(() => ([
        "https://lf3-static.bytednsdoc.com/obj/eden-cn/ptlz_zlp/ljhwZthlaukjlkulzlp/root-web-sites/abstract.jpg",
        "https://lf3-static.bytednsdoc.com/obj/eden-cn/ptlz_zlp/ljhwZthlaukjlkulzlp/root-web-sites/sky.jpg",
        "https://lf3-static.bytednsdoc.com/obj/eden-cn/ptlz_zlp/ljhwZthlaukjlkulzlp/root-web-sites/greenleaf.jpg",
    ]), []);

    const renderPreviewMenu = useCallback((props) => {
        const {
            ratio,
            disabledPrev,
            disabledNext,
            disableZoomIn,
            disableZoomOut,
            disableDownload,
            onDownload,
            onNext,
            onPrev,
            onRotateLeft,
            onRatioClick,
            onZoomIn,
            onZoomOut,
        } = props;
        return (
            <div 
                style={{ 
                    background: "grey", 
                    height: 40, 
                    width: 280, 
                    display: "flex",
                    alignItems: "center",
                    justifyContent: "space-around",
                    borderRadius: 3,
                }}
            >
                <Button
                    icon={<IconCozArrowLeft size="large" />}
                    type="tertiary"
                    onClick={!disabledPrev ? onPrev : undefined}
                    disabled={disabledPrev}
                />
                <Button
                    icon={<IconCozArrowRight size="large" />}
                    type="tertiary"                     
                    onClick={!disabledNext ? onNext : undefined}
                    disabled={disabledNext}
                />
                <Button
                    icon={<IconCozMinus size="large" />}
                    type="tertiary"
                    onClick={!disableZoomOut ? onZoomOut : undefined}
                    disabled={disableZoomOut} 
                />
                <Button
                    icon={<IconCozPlus size="large" />}
                    type="tertiary"
                    onClick={!disableZoomIn ? onZoomIn : undefined} 
                    disabled={disableZoomIn}
                />
                <Button
                    icon={ratio === "adaptation" ? <IconCozOriginalSize size="large" /> : <IconCozAutoView size="large" />}
                    type="tertiary"
                    onClick={onRatioClick} 
                />
                <Button
                    icon={<IconCozRotationFill size="large" />}
                    type="tertiary"
                    onClick={onRotateLeft}
                />
                <Button
                    icon={<IconCozDownload size="large" />}
                    type="tertiary"
                    onClick={!disableDownload ? onDownload : undefined}
                    disabled={disableDownload}
                />
            </div>);
    }, []);


    return ( 
        <ImagePreview renderPreviewMenu={renderPreviewMenu}>
            {srcList.map((src, index) => {
                return (
                    <Image 
                        key={index} 
                        src={src} 
                        width={200} 
                        alt={'lamp' + (index + 1)} 
                        style={{ marginRight: 5 }}
                    />
                );
            })}
        </ImagePreview>
    );
};

export default Demo;
