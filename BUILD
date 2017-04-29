load("@io_bazel_rules_go//go:def.bzl", "go_binary", "go_library", "go_prefix")

go_prefix("kope.io/imagebuilder")

go_library(
    name = "go_default_library",
    srcs = [
        "buffer_byte_source.go",
        "build_context.go",
        "byte_source.go",
        "docker_bundle.go",
        "file_byte_source.go",
        "filedata.go",
        "gzip_byte_source.go",
        "layer.go",
        "main.go",
        "once_byte_source.go",
        "simple_filedata.go",
        "slice_byte_source.go",
        "task_add_deb.go",
        "task_add_tar.go",
        "task_add_tree.go",
        "task_create_group.go",
        "task_create_user.go",
        "utils.go",
        "xz_byte_source.go",
    ],
    visibility = ["//visibility:private"],
    deps = [
        "@com_github_blakesmith_ar//:go_default_library",
        "@com_github_golang_glog//:go_default_library",
        "@org_xi2_x_xz//:go_default_library",
    ],
)

go_binary(
    name = "imagebuilder",
    library = ":go_default_library",
    visibility = ["//visibility:public"],
)
