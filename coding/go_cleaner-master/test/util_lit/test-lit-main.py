#!/usr/bin/env python3
import os
from lit.main import main    

def filter_out_tests():
    excludes = [
        'fn_ptr_field_based',
        'fn_ptr_field_sensitive',
        'fn_ptr_flow_sensitive',
        'fn_reflection_call',
        'var_dead_store',
        'var_dead_value_passing',
        'var_unreachable',
        'codeclean',
        'interface_method',
    ]
    os.environ['LIT_FILTER_OUT'] = '|'.join(excludes)

if __name__ == '__main__':
    filter_out_tests()
    main()