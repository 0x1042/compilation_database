# compile_commands.json for bezel project

copy from https://github.com/redpanda-data/redpanda

# Usage 

## `MODULE.bazel`

```starlark

bazel_dep(name = "compilation_database", version = "0.2.0")

git_override(
    module_name = "compilation_database",
    remote = "https://github.com/0x1042/compilation_database.git",
    commit = "8bd30078d055ce0e2ae2e7cb67214d9961a3d5de",
)
```

## `BUILD.bazel`

```starlark
alias(
    name = "cc_gen",
    actual = "@compilation_database//:compilation_database_v2",
)
```

```shell
bazel run //:cc_gen -- --target="//your_target"
```
