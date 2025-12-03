; TypeScript/TSX query for functions, types, and imports

; Function declarations with parameters
(function_declaration
  name: (identifier) @func.name
  parameters: (formal_parameters) @func.params)

; Arrow functions assigned to variables
(variable_declarator
  name: (identifier) @func.name
  value: (arrow_function
    parameters: (formal_parameters) @func.params))

; Method definitions
(method_definition
  name: (property_identifier) @func.name
  parameters: (formal_parameters) @func.params)

; Interface declarations
(interface_declaration
  name: (type_identifier) @type.name) @type.interface

; Class declarations
(class_declaration
  name: (type_identifier) @type.name) @type.class

; Type aliases
(type_alias_declaration
  name: (type_identifier) @type.name) @type.alias

; Enum declarations
(enum_declaration
  name: (identifier) @type.name) @type.enum

; ES6 imports
(import_statement
  source: (string) @import)
