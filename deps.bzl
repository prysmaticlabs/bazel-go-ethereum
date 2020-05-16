load("@bazel_gazelle//:deps.bzl", _go_repository = "go_repository")

def _maybe(repo_rule, name, **kwargs):
    if name not in native.existing_rules():
        repo_rule(name = name, **kwargs)

def go_repository(name, **kwargs):
    _maybe(_go_repository, name, **kwargs)

def geth_dependencies():

    go_repository(
        name = "com_github_tyler_smith_go_bip39",
        commit = "c55f737395bc849274c1d1a35ca6f29a4e568092",
        importpath = "github.com/tyler-smith/go-bip39",
    )

    go_repository(
        name = "com_github_dop251_goja",
        commit = "77e84ffb8c65af72e4a3bc53c31328265eac7c81",
        importpath = "github.com/dop251/goja",
    )

    go_repository(
        name = "com_github_azure_azure_storage_blob_go",
        importpath = "github.com/Azure/azure-storage-blob-go",
        sum = "h1:53qhf0Oxa0nOjgbDeeYPUeyiNmafAFEY95rZLK0Tj6o=",
        version = "v0.8.0",
    )

    go_repository(
        name = "com_github_wsddn_go_ecdh",
        commit = "48726bab92085232373de4ec5c51ce7b441c63a0",
        importpath = "github.com/wsddn/go-ecdh",
    )

    go_repository(
        name = "com_github_gorilla_websocket",
        commit = "b65e62901fc1c0d968042419e74789f6af455eb9",
        importpath = "github.com/gorilla/websocket",
    )

    go_repository(
        name = "com_github_cloudflare_cloudflare_go",
        commit = "b0c040d69602043dae69d2474368c0dbada65cf3",
        importpath = "github.com/cloudflare/cloudflare-go",
    )

    go_repository(
        name = "com_github_aws_aws_sdk_go",
        commit = "439e0f33d9531b21d122b39c05cb41f30f7df5cf",
        importpath = "github.com/aws/aws-sdk-go",
    )

    go_repository(
        name = "com_github_gballet_go_libpcsclite",
        commit = "4678299bea08415f0ca8bd71da9610625cc86e86",
        importpath = "github.com/gballet/go-libpcsclite",
    )

    go_repository(
        name = "com_github_status_im_keycard_go",
        commit = "957c095369694cc23ce9f2bca4acd325764860eb",
        importpath = "github.com/status-im/keycard-go",
    )

    go_repository(
        name = "com_github_steakknife_bloomfilter",
        commit = "6819c0d2a57025e1700378ee59b7382d71987f14",
        importpath = "github.com/steakknife/bloomfilter",
    )

    go_repository(
        name = "com_github_victoriametrics_fastcache",
        commit = "8835719dc76cc26f97026e3aa726742e7d2f1053",
        importpath = "github.com/VictoriaMetrics/fastcache",
    )

    go_repository(
        name = "com_github_prometheus_tsdb",
        commit = "656e53533ce79e020d44a52b116c9769fc6e681a",
        importpath = "github.com/prometheus/tsdb",
    )

    go_repository(
        name = "com_github_azure_azure_pipeline_go",
        importpath = "github.com/Azure/azure-pipeline-go",
        sum = "h1:6oiIS9yaG6XCCzhgAgKFfIWyo4LLCiDhZot6ltoThhY=",
        version = "v0.2.2",
    )

    go_repository(
        name = "com_github_dlclark_regexp2",
        commit = "5a37df65fb8b90da0c2126cbab004089d3a24390",
        importpath = "github.com/dlclark/regexp2",
    )

    go_repository(
        name = "com_github_mattn_go_ieproxy",
        commit = "439dd0581a2a03b415673a2462ad5c21eaabc588",
        importpath = "github.com/mattn/go-ieproxy",
    )

    go_repository(
        name = "org_golang_x_time",
        commit = "89c76fbcd5d1cd4969e5d2fe19d48b19d5ad94a0",
        importpath = "golang.org/x/time",
    )

    go_repository(
        name = "com_github_go_sourcemap_sourcemap",
        commit = "2588a51d69f1c3e296d57f75815ef40a20d54245",
        importpath = "github.com/go-sourcemap/sourcemap",
    )

    go_repository(
        name = "com_github_cespare_xxhash",
        importpath = "github.com/cespare/xxhash/v2",
        sum = "h1:6MnRN8NT7+YBpUIWxHtefFZOKTAPgGjpQSxqLNn0+qY=",
        version = "v2.1.1",
    )

    go_repository(
        name = "com_github_steakknife_hamming",
        importpath = "github.com/steakknife/hamming",
        sum = "h1:njlZPzLwU639dk2kqnCPPv+wNjq7Xb6EfUxe/oX0/NM=",
        version = "v0.0.0-20180906055917-c99c65617cd3",
    )

    go_repository(
        name = "com_github_jmespath_go_jmespath",
        importpath = "github.com/jmespath/go-jmespath",
        sum = "h1:OS12ieG61fsCg5+qLJ+SsW9NicxNkg3b25OyT2yCeUc=",
        version = "v0.3.0",
    )
