import React, { useState } from 'react';
import { LogicExpr } from '@cozeloop/components';

const Demo = () => {
  const [value, setValue] = useState({
    exprs: [{ left: '', operator: '', right: '' }],
  });

  return (
    <LogicExpr
      defaultExpr={{ left: '', operator: '', right: '' }}
      value={value}
      onChange={setValue}
      leftRender={({ expr, onChange }) => (
        <input
          value={expr.left}
          onChange={(e) => onChange(e.target.value)}
          placeholder="字段"
          style={{ width: 120 }}
        />
      )}
      operatorRender={({ expr, onChange }) => (
        <select value={expr.operator} onChange={(e) => onChange(e.target.value)}>
          <option value="">选择操作符</option>
          <option value="eq">等于</option>
          <option value="neq">不等于</option>
        </select>
      )}
      rightRender={({ expr, onChange }) => (
        <input
          value={expr.right}
          onChange={(e) => onChange(e.target.value)}
          placeholder="值"
          style={{ width: 120 }}
        />
      )}
      allowLogicOperators={['and', 'or']}
      maxNestingDepth={2}
    />
  );
};

export default Demo;
