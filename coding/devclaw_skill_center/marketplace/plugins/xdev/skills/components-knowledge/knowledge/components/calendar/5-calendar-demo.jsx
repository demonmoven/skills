import React from 'react';
import { Calendar } from '@coze-arch/coze-design';

const Demo = () => <Calendar mode="range" range={[new Date(2020, 8, 26), new Date(2020, 8, 31)]} />;

export default Demo;
