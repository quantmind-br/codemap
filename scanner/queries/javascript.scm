; JavaScript/JSX query for functions, classes, and imports

; Function declarations with parameters
(function_declaration
  name: (identifier) @func.name
  parameters: (formal_parameters) @func.params)

; Arrow functions assigned to variables
(variable_declarator
  name: (identifier) @func.name
  value: (arrow_function
    parameters: (formal_parameters) @func.params))

; Method definitions in classes/objects
(method_definition
  name: (property_identifier) @func.name
  parameters: (formal_parameters) @func.params)

; Class declarations
(class_declaration
  name: (identifier) @type.name) @type.class

; ES6 imports: import x from 'y'
(import_statement
  source: (string) @import)

; CommonJS: require('x')
(call_expression
  function: (identifier) @_req (#eq? @_req "require")
  arguments: (arguments (string) @import))
