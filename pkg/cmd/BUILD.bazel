load("@io_bazel_rules_go//go:def.bzl", "go_library")

go_library(
    name = "go_default_library",
    srcs = [
        "copy.go",
        "create.go",
        "create_layer.go",
        "delete.go",
        "delete_layer.go",
        "env.go",
        "factory.go",
        "fetch.go",
        "push.go",
        "root.go",
        "set.go",
    ],
    importpath = "kope.io/imagebuilder/pkg/cmd",
    visibility = ["//visibility:public"],
    deps = [
        "//pkg/docker:go_default_library",
        "//pkg/imageconfig:go_default_library",
        "//pkg/layers:go_default_library",
        "@com_github_golang_glog//:go_default_library",
        "@com_github_spf13_cobra//:go_default_library",
    ],
)
