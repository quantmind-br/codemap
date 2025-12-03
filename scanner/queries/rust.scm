; Rust query for functions, types, and imports

; Function definitions with parameters
(function_item
  name: (identifier) @func.name
  parameters: (parameters) @func.params)

; Struct definitions
(struct_item
  name: (type_identifier) @type.name) @type.struct

; Enum definitions
(enum_item
  name: (type_identifier) @type.name) @type.enum

; Trait definitions
(trait_item
  name: (type_identifier) @type.name) @type.trait

; use statements
(use_declaration
  argument: (scoped_identifier) @import)

(use_declaration
  argument: (identifier) @import)

; mod declarations (internal modules)
(mod_item
  name: (identifier) @module)
