; scripts/l2-graph/tree-sitter-queries/python.scm
(function_definition
  name: (identifier) @name.definition.function) @definition.function

(class_definition
  name: (identifier) @name.definition.class) @definition.class

(call
  function: [
    (identifier) @name.reference.call
    (attribute attribute: (identifier) @name.reference.call)
  ]) @reference.call

(import_statement (dotted_name) @name.reference.import) @reference.import
(import_from_statement module_name: (dotted_name) @name.reference.import) @reference.import
