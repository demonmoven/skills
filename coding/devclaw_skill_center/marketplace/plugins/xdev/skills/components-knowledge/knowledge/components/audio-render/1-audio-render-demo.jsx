import React from 'react';
import { AudioRender } from '@cozeloop/components';

const Demo = () => (
  <AudioRender
    audioUrl="https://example.com/sample-audio.mp3"
    audioName="sample-audio.mp3"
    showDownload={true}
    size="medium"
  />
);

export default Demo;
