; scripts/l2-graph/tree-sitter-queries/go.scm
(function_declaration
  name: (identifier) @name.definition.function) @definition.function

(method_declaration
  name: (field_identifier) @name.definition.method) @definition.method

(type_spec
  name: (type_identifier) @name.definition.type) @definition.type

(call_expression
  function: [
    (identifier) @name.reference.call
    (selector_expression field: (field_identifier) @name.reference.call)
  ]) @reference.call

(import_spec path: (interpreted_string_literal) @name.reference.import) @reference.import
