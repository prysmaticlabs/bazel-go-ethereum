load("@bazel_tools//tools/build_defs/repo:http.bzl", "http_archive")

http_archive(
    name = "io_bazel_rules_go",
    sha256 = "513c12397db1bc9aa46dd62f02dd94b49a9b5d17444d49b5a04c5a89f3053c1c",
    urls = [
        "https://storage.googleapis.com/bazel-mirror/github.com/bazelbuild/rules_go/releases/download/v0.19.5/rules_go-v0.19.5.tar.gz",
        "https://github.com/bazelbuild/rules_go/releases/download/v0.19.5/rules_go-v0.19.5.tar.gz",
    ],
    patch_args = ["-p1"],
    patches = [
            # TODO remove patch once https://github.com/bazelbuild/rules_go/pull/2283 is merged
            # needed to properly handle go_path for libsecp256k1 in bazel-go-ethereum
            "//third_party:io_bazel_rules_go-go_path.patch"
    ],
)

http_archive(
    name = "bazel_gazelle",
    sha256 = "7fc87f4170011201b1690326e8c16c5d802836e3a0d617d8f75c3af2b23180c4",
    urls = [
        "https://storage.googleapis.com/bazel-mirror/github.com/bazelbuild/bazel-gazelle/releases/download/0.18.2/bazel-gazelle-0.18.2.tar.gz",
        "https://github.com/bazelbuild/bazel-gazelle/releases/download/0.18.2/bazel-gazelle-0.18.2.tar.gz",
    ],
)

load("@io_bazel_rules_go//go:deps.bzl", "go_rules_dependencies", "go_register_toolchains")

go_rules_dependencies()

go_register_toolchains()

load("@bazel_gazelle//:deps.bzl", "gazelle_dependencies", "go_repository")

gazelle_dependencies()

go_repository(
    name = "com_github_ethereum_go_ethereum",
    commit = "e1edfe0689966d5b5fcee530a96c31dd28aea95c",
    importpath = "github.com/ethereum/go-ethereum",
)

go_repository(
    name = "org_golang_x_tools",
    commit = "537d06c362073e8c95164d0d4709059603cfdb02",
    importpath = "golang.org/x/tools",
)

go_repository(
    name = "com_github_deckarep_golang_set",
    commit = "699df6a3acf6867538e50931511e9dc403da108a",
    importpath = "github.com/deckarep/golang-set",
)

go_repository(
    name = "com_github_pborman_uuid",
    commit = "8b1b92947f46224e3b97bb1a3a5b0382be00d31e",
    importpath = "github.com/pborman/uuid",
)

go_repository(
    name = "com_github_rjeczalik_notify",
    commit = "629144ba06a1c6af28c1e42c228e3d42594ce081",
    importpath = "github.com/rjeczalik/notify",
)

go_repository(
    name = "org_golang_x_crypto",
    commit = "8dd112bcdc25174059e45e07517d9fc663123347",
    importpath = "golang.org/x/crypto",
)

go_repository(
    name = "com_github_golang_protobuf",
    commit = "1d3f30b51784bec5aad268e59fd3c2fc1c2fe73f",
    importpath = "github.com/golang/protobuf",
)

go_repository(
    name = "in_gopkg_urfave_cli_v1",
    commit = "cfb38830724cc34fedffe9a2a29fb54fa9169cd1",
    importpath = "gopkg.in/urfave/cli.v1",
)

go_repository(
    name = "org_golang_x_net",
    commit = "927f97764cc334a6575f4b7a1584a147864d5723",
    importpath = "golang.org/x/net",
)

go_repository(
    name = "com_github_elastic_gosigar",
    commit = "f498c67133bcded80f5966ee63acfe68cff4e6bf",
    importpath = "github.com/elastic/gosigar",
)

go_repository(
    name = "com_github_naoina_toml",
    commit = "9fafd69674167c06933b1787ae235618431ce87f",
    importpath = "github.com/naoina/toml",
)

go_repository(
    name = "com_github_syndtr_goleveldb",
    commit = "b001fa50d6b27f3f0bb175a87d0cb55426d0a0ae",
    importpath = "github.com/syndtr/goleveldb",
)

go_repository(
    name = "com_github_olekukonko_tablewriter",
    commit = "e6d60cf7ba1f42d86d54cdf5508611c4aafb3970",
    importpath = "github.com/olekukonko/tablewriter",
)

go_repository(
    name = "org_golang_x_sys",
    commit = "8ff4e546d48be377a38a3ab13ef2dd4dd8dce06a",
    importpath = "golang.org/x/sys",
)

go_repository(
    name = "com_github_mattn_go_colorable",
    commit = "efa589957cd060542a26d2dd7832fd6a6c6c3ade",
    importpath = "github.com/mattn/go-colorable",
)

go_repository(
    name = "com_github_mattn_go_isatty",
    commit = "3fb116b820352b7f0c281308a4d6250c22d94e27",
    importpath = "github.com/mattn/go-isatty",
)

go_repository(
    name = "com_github_aristanetworks_goarista",
    commit = "0ca71131d8f76e0739563d97cc9f95c021825a2b",
    importpath = "github.com/aristanetworks/goarista",
)

go_repository(
    name = "com_github_hashicorp_golang_lru",
    commit = "20f1fb78b0740ba8c3cb143a61e86ba5c8669768",
    importpath = "github.com/hashicorp/golang-lru",
)

go_repository(
    name = "com_github_edsrzf_mmap_go",
    commit = "188cc3b666ba704534fa4f96e9e61f21f1e1ba7c",
    importpath = "github.com/edsrzf/mmap-go",
)

go_repository(
    name = "com_github_peterh_liner",
    commit = "5a0dfa99e2aa1d433a9642e863da51402e609376",
    importpath = "github.com/peterh/liner",
)

go_repository(
    name = "com_github_robertkrimen_otto",
    commit = "15f95af6e78dcd2030d8195a138bd88d4f403546",
    importpath = "github.com/robertkrimen/otto",
)

go_repository(
    name = "com_github_mohae_deepcopy",
    commit = "c48cc78d482608239f6c4c92a4abd87eb8761c90",
    importpath = "github.com/mohae/deepcopy",
)

go_repository(
    name = "in_gopkg_olebedev_go_duktape_v3",
    commit = "ccb656ba24c2a193e5650299767e2df1554725fd",
    importpath = "gopkg.in/olebedev/go-duktape.v3",
)

go_repository(
    name = "com_github_golang_snappy",
    commit = "2e65f85255dbc3072edf28d6b5b8efc472979f5a",
    importpath = "github.com/golang/snappy",
)

go_repository(
    name = "com_github_docker_docker",
    commit = "8aca18d631f3f72d4c6e3dc01b6e5d468ad941b8",
    importpath = "github.com/docker/docker",
)

go_repository(
    name = "com_github_fjl_memsize",
    commit = "2a09253e352a56f419bd88effab0483f52da4c7d",
    importpath = "github.com/fjl/memsize",
)

go_repository(
    name = "com_github_davecgh_go_spew",
    commit = "d8f796af33cc11cb798c1aaeb27a4ebc5099927d",
    importpath = "github.com/davecgh/go-spew",
)

go_repository(
    name = "com_github_fatih_color",
    commit = "3f9d52f7176a6927daacff70a3e8d1dc2025c53e",
    importpath = "github.com/fatih/color",
)

go_repository(
    name = "com_github_go_stack_stack",
    commit = "2fee6af1a9795aafbe0253a0cfbdf668e1fb8a9a",
    importpath = "github.com/go-stack/stack",
)

go_repository(
    name = "com_github_influxdata_influxdb",
    commit = "7db3db278bfafee3f10767b9322720035688f18b",
    importpath = "github.com/influxdata/influxdb",
)

go_repository(
    name = "com_github_huin_goupnp",
    commit = "656e61dfadd241c7cbdd22a023fa81ecb6860ea8",
    importpath = "github.com/huin/goupnp",
)

go_repository(
    name = "com_github_jackpal_go_nat_pmp",
    commit = "d89d09f6f3329bc3c2479aa3cafd76a5aa93a35c",
    importpath = "github.com/jackpal/go-nat-pmp",
)

go_repository(
    name = "org_golang_x_text",
    commit = "17bcc049122f272a32787ba38073ee47433023e9",
    importpath = "golang.org/x/text",
)

go_repository(
    name = "com_github_opentracing_opentracing_go",
    commit = "8b5a441dc60e8d9bb9bf9d22060b6d94fa599983",
    importpath = "github.com/opentracing/opentracing-go",
)

go_repository(
    name = "com_github_julienschmidt_httprouter",
    commit = "26a05976f9bf5c3aa992cc20e8588c359418ee58",
    importpath = "github.com/julienschmidt/httprouter",
)

go_repository(
    name = "com_github_rs_cors",
    commit = "a3460e445dd310dbefee993fe449f2ff9c08ae71",
    importpath = "github.com/rs/cors",
)

go_repository(
    name = "org_bazil_fuse",
    commit = "65cc252bf6691cb3c7014bcb2c8dc29de91e3a7e",
    importpath = "bazil.org/fuse",
)

go_repository(
    name = "com_github_pkg_errors",
    commit = "059132a15dd08d6704c67711dae0cf35ab991756",
    importpath = "github.com/pkg/errors",
)

go_repository(
    name = "com_github_uber_jaeger_client_go",
    commit = "68407ce61637ffc3b1c455bd02e25a9da3fb45de",
    importpath = "github.com/uber/jaeger-client-go",
)

go_repository(
    name = "com_github_uber_jaeger_lib",
    commit = "0e30338a695636fe5bcf7301e8030ce8dd2a8530",
    importpath = "github.com/uber/jaeger-lib",
)

go_repository(
    name = "org_golang_x_sync",
    commit = "42b317875d0fa942474b76e1b46a6060d720ae6e",
    importpath = "golang.org/x/sync",
)

go_repository(
    name = "in_gopkg_check_v1",
    commit = "788fd78401277ebd861206a03c884797c6ec5541",
    importpath = "gopkg.in/check.v1",
)

go_repository(
    name = "com_github_stretchr_testify",
    commit = "5b93e2dc01fd8fbf32aa74a198b0ebe78f6f6b6f",
    importpath = "github.com/stretchr/testify",
)

go_repository(
    name = "com_github_kr_pretty",
    commit = "73f6ac0b30a98e433b289500d779f50c1a6f0712",
    importpath = "github.com/kr/pretty",
)

go_repository(
    name = "com_github_kr_text",
    commit = "e2ffdb16a802fe2bb95e2e35ff34f0e53aeef34f",
    importpath = "github.com/kr/text",
)

go_repository(
    name = "com_github_google_uuid",
    commit = "9b3b1e0f5f99ae461456d768e7d301a7acdaa2d8",
    importpath = "github.com/google/uuid",
)

go_repository(
    name = "com_github_cespare_cp",
    commit = "db1407d84ae423533fe1d25510c1c4c4d831f0fc",
    importpath = "github.com/cespare/cp",
)

go_repository(
    name = "com_github_mattn_go_runewidth",
    commit = "703b5e6b11ae25aeb2af9ebb5d5fdf8fa2575211",
    importpath = "github.com/mattn/go-runewidth",
)

go_repository(
    name = "com_github_nsf_termbox_go",
    commit = "60ab7e3d12ed91bc1b2486559c4b3a6b62297577",
    importpath = "github.com/nsf/termbox-go",
)

go_repository(
    name = "in_gopkg_sourcemap_v1",
    commit = "6e83acea0053641eff084973fee085f0c193c61a",
    importpath = "gopkg.in/sourcemap.v1",
)

go_repository(
    name = "com_github_naoina_go_stringutil",
    commit = "6b638e95a32d0c1131db0e7fe83775cbea4a0d0b",
    importpath = "github.com/naoina/go-stringutil",
)

go_repository(
    name = "com_github_mitchellh_go_wordwrap",
    commit = "9e67c67572bc5dd02aef930e2b0ae3c02a4b5a5c",
    importpath = "github.com/mitchellh/go-wordwrap",
)

go_repository(
    name = "com_github_influxdata_platform",
    commit = "e66db43f958eebf221239eb919def7803644115d",
    importpath = "github.com/influxdata/platform",
)

go_repository(
    name = "com_github_btcsuite_btcd",
    commit = "306aecffea325e97f513b3ff0cf7895a5310651d",
    importpath = "github.com/btcsuite/btcd",
)

go_repository(
    name = "com_github_graph_gophers_graphql_go",
    commit = "3e8838d4614c12ab337e796548521744f921e05d",
    importpath = "github.com/graph-gophers/graphql-go",
)

go_repository(
    name = "com_github_allegro_bigcache",
    commit = "e24eb225f15679bbe54f91bfa7da3b00e59b9768",
    importpath = "github.com/allegro/bigcache",
)

go_repository(
    name = "com_github_gizak_termui",
    commit = "d5ea67dfda30b3881742402ad785ec770e066996",
    importpath = "github.com/gizak/termui",
)

go_repository(
    name = "com_github_cjbassi_drawille_go",
    commit = "27dc511fe6fd820bf8537298e1d447b450210b28",
    importpath = "github.com/cjbassi/drawille-go",
)

go_repository(
    name = "in_gopkg_natefinch_npipe_v2",
    commit = "c1b8fa8bdccecb0b8db834ee0b92fdbcfa606dd6",
    importpath = "gopkg.in/natefinch/npipe.v2",
)
