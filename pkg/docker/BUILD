load("@io_bazel_rules_go//go:def.bzl", "go_library")

go_library(
    name = "go_default_library",
    srcs = [
        "auth.go",
        "registry.go",
    ],
    importpath = "kope.io/imagebuilder/pkg/docker",
    visibility = ["//visibility:public"],
    deps = ["@com_github_golang_glog//:go_default_library"],
)
