; C# query for methods, types, and using directives

; Method declarations with parameters
(method_declaration
  name: (identifier) @func.name
  parameters: (parameter_list) @func.params)

; Constructor declarations
(constructor_declaration
  name: (identifier) @func.name
  parameters: (parameter_list) @func.params)

; Class declarations
(class_declaration
  name: (identifier) @type.name) @type.class

; Interface declarations
(interface_declaration
  name: (identifier) @type.name) @type.interface

; Struct declarations
(struct_declaration
  name: (identifier) @type.name) @type.struct

; Enum declarations
(enum_declaration
  name: (identifier) @type.name) @type.enum

; Using directives
(using_directive
  (qualified_name) @import)

(using_directive
  (identifier) @import)
