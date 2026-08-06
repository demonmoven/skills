import React from 'react';
import { VideoRender } from '@cozeloop/components';

const Demo = () => (
  <VideoRender
    videoUrl="https://example.com/sample-video.mp4"
    videoName="sample-video.mp4"
    showDownload={true}
    size="medium"
  />
);

export default Demo;
