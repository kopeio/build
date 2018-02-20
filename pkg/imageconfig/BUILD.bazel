load("@io_bazel_rules_go//go:def.bzl", "go_library")

go_library(
    name = "go_default_library",
    srcs = ["config.go"],
    importpath = "kope.io/imagebuilder/pkg/imageconfig",
    visibility = ["//visibility:public"],
    deps = ["//pkg/layers:go_default_library"],
)
