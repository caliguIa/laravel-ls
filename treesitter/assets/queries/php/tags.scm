; --------------------------------------------------
;  Config
; --------------------------------------------------

; config() calls
(function_call_expression
  function: (name) @function (#eq? @function "config")
  arguments: (arguments
    . (argument [
       (string (string_content)?) @config.key
       (encapsed_string . (string_content) .) @config.key
       (encapsed_string "\"" . "\"") @config.key
    ])
  ))

; config()->type() calls
(member_call_expression
  object: (
    function_call_expression
    function: (name) @object.name (#eq? @object.name "config")
    arguments: (arguments "(" . ")"))
  name: (name) @function.name
  (#any-of? @function.name "get" "integer" "float" "string" "boolean" "array")
  arguments: (arguments
     . (argument [
        (string (string_content)?) @config.key
        (encapsed_string . (string_content) .) @config.key
        (encapsed_string "\"" . "\"") @config.key
     ])
  ))

; Config::type() calls.
(scoped_call_expression
  scope: [
    (qualified_name (name) @class)
    (name) @class
  ] (#eq? @class "Config")
  name: (name) @method
  (#any-of? @method "get" "integer" "float" "string" "boolean" "array")
  arguments: (arguments
     . (argument [
        (string (string_content)?) @config.key
        (encapsed_string . (string_content) .) @config.key
        (encapsed_string "\"" . "\"") @config.key
     ])
  ))

; Config::getMany() calls.
(scoped_call_expression
  scope: [
    (qualified_name (name) @class)
    (name) @class
  ] (#eq? @class "Config")
  name: (name) @method (#eq? @method "getMany")
  arguments: (arguments
    . (argument
        (array_creation_expression
          (array_element_initializer
            [
              (string (string_content)?) @config.key
              (encapsed_string . (string_content) .) @config.key
              (encapsed_string "\"" . "\"") @config.key
            ]
    )))
  ))

; config()->getMany() calls.
(member_call_expression
  object: (
    function_call_expression
    function: (name) @object.name (#eq? @object.name "config")
    arguments: (arguments "(" . ")"))
  name: (name) @function.name (#eq? @function.name "getMany")
  arguments: (arguments
    . (argument
        (array_creation_expression
          (array_element_initializer
            [
             (string (string_content)?) @config.key
             (encapsed_string . (string_content) .) @config.key
             (encapsed_string "\"" . "\"") @config.key
             ]
    )))
  ))

; --------------------------------------------------
;  App
; --------------------------------------------------

; app('key')
(function_call_expression
  function: (name) @function (#eq? @function "app")
  arguments: (arguments
    . (argument [
        (string (string_content)?) @app.service
        (encapsed_string . (string_content) .) @app.service
        (encapsed_string "\"" . "\"") @app.service
    ])
  ))


; app()->make('key')
; app()->bound('key')
; app()->isShared('key')
(member_call_expression
  object: (
    function_call_expression
    function: (name) @object.name (#eq? @object.name "app")
    arguments: (arguments "(" . ")"))
  name: (name) @function.name
  (#any-of? @function.name "make" "bound" "isShared")
  arguments: (arguments
    . (argument [
        (string (string_content)?) @app.service
        (encapsed_string . (string_content) .) @app.service
        (encapsed_string "\"" . "\"") @app.service
    ])
  ))

; App::make('key')
; App::bound('key')
; App::isShared('key')
(scoped_call_expression
  scope: [
    (qualified_name (name) @class)
    (name) @class
  ] (#eq? @class "App")
  name: (name) @method
  (#any-of? @method "make" "bound" "isShared")
  arguments: (arguments
    . (argument [
        (string (string_content)?) @app.service
        (encapsed_string . (string_content) .) @app.service
        (encapsed_string "\"" . "\"") @app.service
    ])
  ))

; --------------------------------------------------
;  View
; --------------------------------------------------

; view() calls
(function_call_expression
  function: (name) @function (#eq? @function "view")
  arguments: (arguments
    . (argument [
        (string (string_content)?) @view.name
        (encapsed_string . (string_content) .) @view.name
        (encapsed_string "\"" . "\"") @view.name
    ])
  ))

; Route::view() calls.
(scoped_call_expression
  scope: [
    (qualified_name (name) @class)
    (name) @class
  ] (#eq? @class "Route")
  name: (name) @method (#eq? @method "view")
  arguments: (arguments
    (argument) ; First parameter is the route.
    . (argument [
        (string (string_content)?) @view.name
        (encapsed_string . (string_content) .) @view.name
        (encapsed_string "\"" . "\"") @view.name
    ])
  ))

; response()->view() calls
(member_call_expression
  object: (
    function_call_expression
    function: (name) @object.name (#eq? @object.name "response")
    arguments: (arguments "(" . ")"))
  name: (name) @function.name
  (#eq? @function.name "view")
  arguments: (arguments
    . (argument [
        (string (string_content)?) @view.name
        (encapsed_string . (string_content) .) @view.name
        (encapsed_string "\"" . "\"") @view.name
    ])
  ))

; --------------------------------------------------
;  Environment
; --------------------------------------------------

; env() calls
(function_call_expression
  function: (name) @function (#eq? @function "env")
  arguments: (arguments
    . (argument [
        (string (string_content)?) @env.key
        (encapsed_string . (string_content) .) @env.key
        (encapsed_string "\"" . "\"") @env.key
    ])
))

; Env::get() calls.
(scoped_call_expression
  scope: [
    (qualified_name (name) @class)
    (name) @class
  ] (#eq? @class "Env")
  name: (name) @method (#eq? @method "get")
  arguments: (arguments
    . (argument [
        (string (string_content)?) @env.key
        (encapsed_string . (string_content) .) @env.key
        (encapsed_string "\"" . "\"") @env.key
    ])
))

; --------------------------------------------------
;  Assets
; --------------------------------------------------

; URL::asset() calls.
(scoped_call_expression
  scope: [
    (qualified_name (name) @class)
    (name) @class
  ] (#eq? @class "URL")
  name: (name) @method (#eq? @method "asset")
  arguments: (arguments
    . (argument [
          (string (string_content)?) @asset.filename
          (encapsed_string (string_content)?) @asset.filename
      ])
))

; asset() calls
(function_call_expression
  function: (name) @function (#eq? @function "asset")
  arguments: (arguments
    . (argument [
        (string (string_content)?) @asset.filename
        (encapsed_string (string_content)?) @asset.filename
    ])
  ))

; --------------------------------------------------
;  Routes
; --------------------------------------------------

; route(), signedRoute(), to_route() calls
(function_call_expression
    function: (name) @function
        (#any-of? @function "route" "signedRoute" "to_route")
    arguments: (arguments
        . (argument [
                (string (string_content)?) @route.name
                (encapsed_string . (string_content) .) @route.name
                (encapsed_string "\"" . "\"") @route.name
        ])
    )
)

; --------------------------------------------------
;  Eloquent
; --------------------------------------------------

; Model::where('column', ...) and other column-accepting query builder methods
; Also matches self::where(...) / static::where(...) inside model classes.
; Emits @eloquent.model (the class name or relative_scope) and @eloquent.column (the first string arg)
(scoped_call_expression
  scope: [
    (qualified_name (name) @eloquent.model)
    (name) @eloquent.model
    (relative_scope) @eloquent.model
  ]
  name: (name) @_method
  (#any-of? @_method
    "where" "whereNot" "orWhere"
    "whereIn" "whereNotIn" "orWhereIn" "orWhereNotIn"
    "whereNull" "whereNotNull" "orWhereNull" "orWhereNotNull"
    "whereBetween" "whereNotBetween" "orWhereBetween"
    "whereDate" "whereMonth" "whereDay" "whereYear" "whereTime"
    "whereColumn" "orWhereColumn"
    "orderBy" "orderByDesc"
    "select" "value" "pluck"
    "sum" "avg" "min" "max" "count")
  arguments: (arguments
    . (argument [
        (string (string_content)?) @eloquent.column
        (encapsed_string . (string_content) .) @eloquent.column
        (encapsed_string "\"" . "\"") @eloquent.column
    ])
  ))

; Chained ->where('column', ...) and other column-accepting query builder methods.
; No @eloquent.model here — the model is resolved in Go by walking up the call chain.
(member_call_expression
  name: (name) @_method
  (#any-of? @_method
    "where" "whereNot" "orWhere"
    "whereIn" "whereNotIn" "orWhereIn" "orWhereNotIn"
    "whereNull" "whereNotNull" "orWhereNull" "orWhereNotNull"
    "whereBetween" "whereNotBetween" "orWhereBetween"
    "whereDate" "whereMonth" "whereDay" "whereYear" "whereTime"
    "whereColumn" "orWhereColumn"
    "orderBy" "orderByDesc"
    "select" "value" "pluck"
    "sum" "avg" "min" "max" "count")
  arguments: (arguments
    . (argument [
        (string (string_content)?) @eloquent.column
        (encapsed_string . (string_content) .) @eloquent.column
        (encapsed_string "\"" . "\"") @eloquent.column
    ])
  ))

; Model::with('relationship', ...) and other relationship-accepting methods
; Also matches self::with(...) / static::with(...) inside model classes.
; Emits @eloquent.model and @eloquent.relationship
(scoped_call_expression
  scope: [
    (qualified_name (name) @eloquent.model)
    (name) @eloquent.model
    (relative_scope) @eloquent.model
  ]
  name: (name) @_method
  (#any-of? @_method
    "with" "without"
    "has" "whereHas" "orWhereHas" "whereDoesntHave")
  arguments: (arguments
    . (argument [
        (string (string_content)?) @eloquent.relationship
        (encapsed_string . (string_content) .) @eloquent.relationship
        (encapsed_string "\"" . "\"") @eloquent.relationship
    ])
  ))

; Chained ->with('relationship', ...) and other relationship-accepting methods.
; No @eloquent.model here — resolved in Go by walking up the call chain.
(member_call_expression
  name: (name) @_method
  (#any-of? @_method
    "with" "without"
    "has" "whereHas" "orWhereHas" "whereDoesntHave")
  arguments: (arguments
    . (argument [
        (string (string_content)?) @eloquent.relationship
        (encapsed_string . (string_content) .) @eloquent.relationship
        (encapsed_string "\"" . "\"") @eloquent.relationship
    ])
  ))

; Model::scopeName() — static scope call
; Also matches self::scopeName() / static::scopeName() inside model classes.
; Emits @eloquent.model and @eloquent.scope (the method name identifier)
(scoped_call_expression
  scope: [
    (qualified_name (name) @eloquent.model)
    (name) @eloquent.model
    (relative_scope) @eloquent.model
  ]
  name: (name) @eloquent.scope
  (#not-any-of? @eloquent.scope
    "where" "whereNot" "orWhere"
    "whereIn" "whereNotIn" "orWhereIn" "orWhereNotIn"
    "whereNull" "whereNotNull" "orWhereNull" "orWhereNotNull"
    "whereBetween" "whereNotBetween" "orWhereBetween"
    "whereDate" "whereMonth" "whereDay" "whereYear" "whereTime"
    "whereColumn" "orWhereColumn"
    "orderBy" "orderByDesc"
    "select" "value" "pluck"
    "sum" "avg" "min" "max" "count"
    "with" "without"
    "has" "whereHas" "orWhereHas" "whereDoesntHave"
    "find" "findOrFail" "findOrNew" "findMany"
    "first" "firstOrFail" "firstOrCreate" "firstOrNew"
    "create" "forceCreate" "insert" "update" "delete"
    "all" "get" "paginate" "simplePaginate" "cursorPaginate"
    "chunk" "chunkById" "each" "eachById"
    "latest" "oldest" "inRandomOrder"
    "withTrashed" "onlyTrashed" "withoutTrashed" "restore" "forceDelete"
    "lockForUpdate" "sharedLock"
    "make" "newQuery" "query" "on" "onWriteConnection"
    "join" "leftJoin" "rightJoin" "crossJoin"
    "groupBy" "having" "havingRaw" "orHaving"
    "limit" "take" "skip" "offset"
    "distinct" "addSelect" "selectRaw"
    "whereRaw" "orWhereRaw"
    "when" "unless" "tap" "pipe"
    "toSql" "toRawSql" "dd" "dump"
    "increment" "decrement" "touch"
    "replicate" "fill" "forceFill" "save" "saveQuietly"
    "fresh" "refresh" "push" "pull"
    "observe" "boot" "booted" "creating" "created" "updating" "updated" "saving" "saved" "deleting" "deleted"
    "upsert" "updateOrCreate" "insertOrIgnore" "insertGetId"
    "doesntHave" "withAggregate" "withCount" "withSum" "withAvg" "withMin" "withMax" "withExists"
    "has" "orHas"))

; Chained ->scopeName() — instance scope call on a query builder chain.
; No @eloquent.model here — resolved in Go by walking up the call chain.
(member_call_expression
  name: (name) @eloquent.scope
  (#not-any-of? @eloquent.scope
    "where" "whereNot" "orWhere"
    "whereIn" "whereNotIn" "orWhereIn" "orWhereNotIn"
    "whereNull" "whereNotNull" "orWhereNull" "orWhereNotNull"
    "whereBetween" "whereNotBetween" "orWhereBetween"
    "whereDate" "whereMonth" "whereDay" "whereYear" "whereTime"
    "whereColumn" "orWhereColumn"
    "orderBy" "orderByDesc"
    "select" "value" "pluck"
    "sum" "avg" "min" "max" "count"
    "with" "without"
    "has" "whereHas" "orWhereHas" "whereDoesntHave"
    "find" "findOrFail" "findOrNew" "findMany"
    "first" "firstOrFail" "firstOrCreate" "firstOrNew"
    "create" "forceCreate" "insert" "update" "delete"
    "all" "get" "paginate" "simplePaginate" "cursorPaginate"
    "chunk" "chunkById" "each" "eachById"
    "latest" "oldest" "inRandomOrder"
    "withTrashed" "onlyTrashed" "withoutTrashed" "restore" "forceDelete"
    "lockForUpdate" "sharedLock"
    "make" "newQuery" "query" "on" "onWriteConnection"
    "join" "leftJoin" "rightJoin" "crossJoin"
    "groupBy" "having" "havingRaw" "orHaving"
    "limit" "take" "skip" "offset"
    "distinct" "addSelect" "selectRaw"
    "whereRaw" "orWhereRaw"
    "when" "unless" "tap" "pipe"
    "toSql" "toRawSql" "dd" "dump"
    "increment" "decrement" "touch"
    "replicate" "fill" "forceFill" "save" "saveQuietly"
    "fresh" "refresh" "push" "pull"
    "observe" "boot" "booted" "creating" "created" "updating" "updated" "saving" "saved" "deleting" "deleted"
    "upsert" "updateOrCreate" "insertOrIgnore" "insertGetId"
    "doesntHave" "withAggregate" "withCount" "withSum" "withAvg" "withMin" "withMax" "withExists"
    "has" "orHas"))
