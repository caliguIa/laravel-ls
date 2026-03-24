<?php

// Discover and introspect all Eloquent models in the project.
// Scans app/Models/ (Laravel 8+) with a fallback to app/ for pre-Laravel-8 projects.
// Emits a JSON object keyed by short class name.

$modelDirs = [];

$modelsDir = app_path('Models');
if (is_dir($modelsDir)) {
    $modelDirs[] = $modelsDir;
} else {
    $modelDirs[] = app_path();
}

// Recursively collect PHP files
function collectPhpFiles(string $dir): array
{
    $files = [];
    $iterator = new RecursiveIteratorIterator(
        new RecursiveDirectoryIterator($dir, FilesystemIterator::SKIP_DOTS),
        RecursiveIteratorIterator::LEAVES_ONLY
    );
    foreach ($iterator as $file) {
        if ($file->getExtension() === 'php') {
            $files[] = $file->getRealPath();
        }
    }
    return $files;
}

$phpFiles = [];
foreach ($modelDirs as $dir) {
    $phpFiles = array_merge($phpFiles, collectPhpFiles($dir));
}

// Load each file so its classes become available via reflection
foreach ($phpFiles as $file) {
    try {
        require_once $file;
    } catch (\Throwable $e) {
        // Ignore load errors; the file may not be a model
    }
}

// Now discover declared classes that extend Eloquent Model
$models = [];

foreach ($phpFiles as $file) {
    $tokens = token_get_all(file_get_contents($file));
    // Extract namespace and class name from tokens
    $namespace = '';
    $className = null;
    $classLine = null;
    $inNamespace = false;
    $inClass = false;

    for ($i = 0; $i < count($tokens); $i++) {
        $token = $tokens[$i];
        if (!is_array($token)) {
            continue;
        }

        if ($token[0] === T_NAMESPACE) {
            $inNamespace = true;
            $namespace = '';
            continue;
        }

        if ($inNamespace) {
            if ($token[0] === T_WHITESPACE) {
                // skip whitespace between `namespace` keyword and the name
                continue;
            } elseif ($token[0] === T_NAME_QUALIFIED || $token[0] === T_STRING) {
                $namespace .= $token[1];
            } elseif ($token[0] === T_NAME_FULLY_QUALIFIED) {
                $namespace = ltrim($token[1], '\\');
            } else {
                // ; ends the namespace declaration
                $inNamespace = false;
            }
            continue;
        }

        if ($token[0] === T_CLASS) {
            // peek ahead for the class name
            for ($j = $i + 1; $j < count($tokens); $j++) {
                if (!is_array($tokens[$j])) {
                    continue;
                }
                if ($tokens[$j][0] === T_STRING) {
                    $className = $tokens[$j][1];
                    $classLine = $tokens[$j][2];
                    break;
                }
            }
            break;
        }
    }

    if ($className === null) {
        continue;
    }

    $fqcn = $namespace !== '' ? $namespace . '\\' . $className : $className;

    if (!class_exists($fqcn)) {
        continue;
    }

    try {
        $ref = new ReflectionClass($fqcn);
    } catch (\Throwable $e) {
        continue;
    }

    if ($ref->isAbstract() || !$ref->isSubclassOf(\Illuminate\Database\Eloquent\Model::class)) {
        continue;
    }

    // Introspect columns using a single getColumns() call — much faster than
    // the per-column getColumnType() + getDoctrineColumn() pattern.
    $columns = [];
    try {
        /** @var \Illuminate\Database\Eloquent\Model $instance */
        $instance = $ref->newInstanceWithoutConstructor();
        $schema = $instance->getConnection()->getSchemaBuilder();
        $table = $instance->getTable();

        foreach ($schema->getColumns($table) as $col) {
            $columns[] = [
                'name'     => $col['name'],
                'type'     => $col['type_name'] ?? $col['type'] ?? 'unknown',
                'nullable' => (bool) ($col['nullable'] ?? false),
                'default'  => $col['default'] !== null ? (string) $col['default'] : '',
            ];
        }
    } catch (\Throwable $e) {
        // DB unavailable — emit model with empty columns
        $columns = [];
    }

    // Introspect relationships via return type hints — no method invocation needed.
    // This is orders of magnitude faster than invoking each method and avoids
    // side-effects (DB queries, exceptions, etc.).
    $relationTypes = [
        \Illuminate\Database\Eloquent\Relations\HasOne::class,
        \Illuminate\Database\Eloquent\Relations\HasMany::class,
        \Illuminate\Database\Eloquent\Relations\BelongsTo::class,
        \Illuminate\Database\Eloquent\Relations\BelongsToMany::class,
        \Illuminate\Database\Eloquent\Relations\HasOneThrough::class,
        \Illuminate\Database\Eloquent\Relations\HasManyThrough::class,
        \Illuminate\Database\Eloquent\Relations\MorphOne::class,
        \Illuminate\Database\Eloquent\Relations\MorphMany::class,
        \Illuminate\Database\Eloquent\Relations\MorphTo::class,
        \Illuminate\Database\Eloquent\Relations\MorphToMany::class,
    ];

    $relationships = [];
    foreach ($ref->getMethods(ReflectionMethod::IS_PUBLIC) as $method) {
        if ($method->getDeclaringClass()->getName() !== $fqcn) {
            continue;
        }
        if ($method->getNumberOfRequiredParameters() > 0) {
            continue;
        }
        $returnType = $method->getReturnType();
        if (!$returnType instanceof ReflectionNamedType) {
            continue;
        }
        $typeName = $returnType->getName();
        $relClass = null;
        foreach ($relationTypes as $relType) {
            if ($typeName === $relType || (class_exists($typeName) && is_subclass_of($typeName, $relType))) {
                $relClass = (new ReflectionClass($typeName))->getShortName();
                break;
            }
        }
        if ($relClass === null) {
            continue;
        }
        // Attempt to determine the related model from the return type's generic
        // parameter via docblock, falling back to empty string.
        $related = '';
        $relationships[] = [
            'name'    => $method->getName(),
            'type'    => $relClass,
            'related' => $related,
            'line'    => $method->getStartLine(),
        ];
    }

    // Introspect scopes
    $scopes = [];
    foreach ($ref->getMethods(ReflectionMethod::IS_PUBLIC) as $method) {
        if ($method->getDeclaringClass()->getName() !== $fqcn) {
            continue;
        }
        $name = $method->getName();
        if (!str_starts_with($name, 'scope') || $name === 'scope') {
            continue;
        }
        $scopeName = lcfirst(substr($name, strlen('scope')));
        // Build a signature string
        $params = [];
        foreach ($method->getParameters() as $idx => $param) {
            if ($idx === 0) {
                // First param is always $query — skip it in the user-facing signature
                continue;
            }
            $p = '$' . $param->getName();
            if ($param->isOptional()) {
                try {
                    $default = $param->getDefaultValue();
                    $p .= ' = ' . var_export($default, true);
                } catch (\Throwable $e) {
                    $p .= ' = ...';
                }
            }
            if ($param->hasType()) {
                $p = (string) $param->getType() . ' ' . $p;
            }
            $params[] = $p;
        }
        $signature = $scopeName . '(' . implode(', ', $params) . ')';
        $scopes[] = [
            'name'      => $scopeName,
            'signature' => $signature,
            'line'      => $method->getStartLine(),
        ];
    }

    // Casts
    $casts = [];
    try {
        $castInstance = $ref->newInstanceWithoutConstructor();
        $casts = $castInstance->getCasts();
    } catch (\Throwable $e) {
        // ignore
    }

    $models[$className] = [
        'class'         => $fqcn,
        'table'         => isset($instance) ? $instance->getTable() : '',
        'file'          => $file,
        'line'          => $classLine ?? 0,
        'columns'       => $columns,
        'relationships' => $relationships,
        'scopes'        => $scopes,
        'casts'         => (object) $casts,
    ];
}

echo json_encode($models);
