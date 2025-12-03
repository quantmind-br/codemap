; Java query for functions, types, and imports

; Method declarations with parameters
(method_declaration
  name: (identifier) @func.name
  parameters: (formal_parameters) @func.params)

; Constructor declarations
(constructor_declaration
  name: (identifier) @func.name
  parameters: (formal_parameters) @func.params)

; Class declarations
(class_declaration
  name: (identifier) @type.name) @type.class

; Interface declarations
(interface_declaration
  name: (identifier) @type.name) @type.interface

; Enum declarations
(enum_declaration
  name: (identifier) @type.name) @type.enum

; Import declarations
(import_declaration
  (scoped_identifier) @import)
