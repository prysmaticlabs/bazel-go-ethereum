load("@bazel_tools//tools/build_defs/repo:http.bzl", "http_archive")

http_archive(
    name = "io_bazel_rules_go",
    sha256 = "207fad3e6689135c5d8713e5a17ba9d1290238f47b9ba545b63d9303406209c6",
    urls = [
        "https://mirror.bazel.build/github.com/bazelbuild/rules_go/releases/download/v0.24.7/rules_go-v0.24.7.tar.gz",
        "https://github.com/bazelbuild/rules_go/releases/download/v0.24.7/rules_go-v0.24.7.tar.gz",
    ],
)

http_archive(
    name = "bazel_gazelle",
    sha256 = "1f4fc1d91826ec436ae04833430626f4cc02c20bb0a813c0c2f3c4c421307b1d",
    strip_prefix = "bazel-gazelle-e368a11b76e92932122d824970dc0ce5feb9c349",
    urls = [
        "https://github.com/bazelbuild/bazel-gazelle/archive/e368a11b76e92932122d824970dc0ce5feb9c349.tar.gz",
    ],
)

load("@io_bazel_rules_go//go:deps.bzl", "go_rules_dependencies", "go_register_toolchains")

go_rules_dependencies()

go_register_toolchains()

load("@bazel_gazelle//:deps.bzl", "gazelle_dependencies", "go_repository")

gazelle_dependencies()

load("//:deps.bzl", "geth_dependencies")

# gazelle:repository_macro deps.bzl%geth_dependencies
geth_dependencies()

