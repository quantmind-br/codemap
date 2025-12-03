; Go query for extracting functions, types, and imports

; Function declarations with parameters
(function_declaration
  name: (identifier) @func.name
  parameters: (parameter_list) @func.params)

; Method declarations (functions with receivers)
(method_declaration
  receiver: (parameter_list) @func.receiver
  name: (field_identifier) @func.name
  parameters: (parameter_list) @func.params)

; Struct type definitions
(type_declaration
  (type_spec
    name: (type_identifier) @type.name
    type: (struct_type))) @type.struct

; Interface type definitions
(type_declaration
  (type_spec
    name: (type_identifier) @type.name
    type: (interface_type))) @type.interface

; Import paths
(import_spec
  path: (interpreted_string_literal) @import)
