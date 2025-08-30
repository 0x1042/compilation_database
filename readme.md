# compile_commands.json for bezel project

copy from https://github.com/redpanda-data/redpanda

# Usage 

## `MODULE.bazel`

```starlark

bazel_dep(name = "compilation_database", version = "0.1.0")

git_override(
    module_name = "compilation_database",
    remote = "https://github.com/0x1042/compilation_database.git",
    tag = "v0.1.0",
)
```

## `BUILD.bazel`

```starlark
alias(
    name = "cc_gen",
    actual = "@compilation_database//:compile_commands",
)
```

```shell
bazel run //:cc_gen -- --target="//your_target"
```