load("@io_bazel_rules_go//go:def.bzl", "go_binary", "go_library")

go_binary(
    name = "kcb",
    importpath = "kope.io/imagebuilder/cmd/kcb",
    library = ":go_default_library",
    visibility = ["//visibility:public"],
)

go_library(
    name = "go_default_library",
    srcs = ["main.go"],
    importpath = "kope.io/imagebuilder/cmd/kcb",
    visibility = ["//visibility:private"],
    deps = ["//pkg/cmd:go_default_library"],
)
