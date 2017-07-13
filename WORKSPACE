git_repository(
    name = "io_bazel_rules_go",
    remote = "https://github.com/bazelbuild/rules_go.git",
    tag = "0.5.0",
)

load("@io_bazel_rules_go//go:def.bzl", "go_repositories", "go_repository", "new_go_repository")

go_repositories()

new_go_repository(
    name = "com_github_golang_glog",
    commit = "23def4e6c14b4da8ac2ed8007337bc5eb5007998",
    importpath = "github.com/golang/glog",
)

new_go_repository(
    name = "com_github_spf13_cobra",
    commit = "7b1b6e8dc027253d45fc029bc269d1c019f83a34",
    importpath = "github.com/spf13/cobra",
)

new_go_repository(
    name = "com_github_spf13_pflag",
    commit = "f1d95a35e132e8a1868023a08932b14f0b8b8fcb",
    importpath = "github.com/spf13/pflag",
)
