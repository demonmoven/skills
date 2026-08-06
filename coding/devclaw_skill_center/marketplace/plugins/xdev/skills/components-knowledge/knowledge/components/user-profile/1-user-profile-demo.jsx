import React from 'react';
import { UserProfile } from '@cozeloop/components';

const Demo = () => (
  <div style={{ display: 'flex', flexDirection: 'column', gap: 12, width: 200 }}>
    <UserProfile
      avatarUrl="https://example.com/avatar.png"
      name="张三"
    />
    <UserProfile name="李四" />
  </div>
);

export default Demo;
