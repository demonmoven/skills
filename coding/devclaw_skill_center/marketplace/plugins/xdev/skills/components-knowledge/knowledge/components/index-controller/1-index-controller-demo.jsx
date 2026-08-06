import React, { useState } from 'react';
import { IndexControllerView } from '@cozeloop/components';

const Demo = () => {
  const [currentIndex, setCurrentIndex] = useState(0);
  const total = 10;

  const indexControllerStore = {
    hasPrevious: currentIndex > 0,
    hasNext: currentIndex < total - 1,
    currentIndex,
    total,
    loading: false,
    goToPrevious: () => setCurrentIndex((prev) => Math.max(0, prev - 1)),
    goToNext: () => setCurrentIndex((prev) => Math.min(total - 1, prev + 1)),
  };

  return <IndexControllerView indexControllerStore={indexControllerStore} />;
};

export default Demo;
