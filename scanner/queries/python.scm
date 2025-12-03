; Python query for extracting functions, classes, and imports

; Function definitions with parameters
(function_definition
  name: (identifier) @func.name
  parameters: (parameters) @func.params)

; Class definitions
(class_definition
  name: (identifier) @type.name) @type.class

; import x, import x.y.z
(import_statement
  name: (dotted_name) @import)

; from x import y
(import_from_statement
  module_name: (dotted_name) @import)

; from x import y, z (the module part)
(import_from_statement
  module_name: (relative_import) @import)
